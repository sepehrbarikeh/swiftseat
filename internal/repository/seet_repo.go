package repository

import (
	"net/http"
	"time"

	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (p *PostgresDB) ReserveSeatWithLock(seatNumber string, eventID uint, userID uint, duration time.Duration) *apperrors.AppError {
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
		var status models.SeatStatus

		var seat models.Seat
		if err := tx.Where("event_id = ? AND seat_number = ?", eventID, seatNumber).First(&seat).Error; err != nil {
			return err
		}

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id = ? AND event_id = ?", seat.ID, eventID).
			First(&status).Error

		if err != nil {
			return err
		}

		now := time.Now()
		if status.Status == "sold" {
			return apperrors.New(http.StatusConflict, "This seat has already been sold", nil)
		}
		if status.Status == "reserved" && status.ExpiresAt != nil && status.ExpiresAt.After(now) {
			return apperrors.New(http.StatusConflict, "This seat is already reserved by another user", nil)
		}

		// reserve and lock for new user
		expiration := now.Add(duration)
		status.Status = "reserved"
		status.ReservedBy = &userID
		status.ExpiresAt = &expiration

		if err := tx.Save(&status).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		// if err already an AppError, return it
		if ae, ok := err.(*apperrors.AppError); ok {
			return ae
		}
		return apperrors.New(http.StatusInternalServerError, "Failed to reserve seat", err)
	}
	return nil
}

// CleanupExpiredSeats
func (p *PostgresDB) CleanupExpiredSeats() (int64, *apperrors.AppError) {
	now := time.Now()

	result := p.DB.Model(&models.SeatStatus{}).
		Where("status = ? AND expires_at < ?", "reserved", now).
		Updates(map[string]interface{}{
			"status":      "available",
			"reserved_by": nil,
			"expires_at":  nil,
		})

	if result.Error != nil {
		return 0, apperrors.New(http.StatusInternalServerError, "Failed to cleanup expired seats", result.Error)
	}

	return result.RowsAffected, nil

}

func (p *PostgresDB) ExecutePaymentTransaction(seatNumber string, eventID, userID uint, amount int64, ticketRef string) (*models.Ticket, *apperrors.AppError) {
	var ticket models.Ticket

	err := p.DB.Transaction(func(tx *gorm.DB) error {
		var status models.SeatStatus

		var seat models.Seat
		if err := tx.Where("event_id = ? AND seat_number = ?", eventID, seatNumber).First(&seat).Error; err != nil {
			return apperrors.New(http.StatusNotFound, "Seat not found", err)
		}

	
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id = ? AND event_id = ?", seat.ID, eventID).
			First(&status).Error
		if err != nil {
			return apperrors.New(http.StatusNotFound, "Seat not found", err)
		}

		// ۲. اعتبارسنجی وضعیت رزرو صندلی
		if status.Status != "reserved" || status.ReservedBy == nil || *status.ReservedBy != userID {
			return apperrors.New(http.StatusBadRequest, "This seat is not reserved by you or has already been sold", nil)
		}
		if status.ExpiresAt != nil && status.ExpiresAt.Before(time.Now()) {
			return apperrors.New(http.StatusGone, "The reservation time limit has expired", nil)
		}


		status.Status = "sold"
		status.ExpiresAt = nil
		if err := tx.Save(&status).Error; err != nil {
			return err
		}

		
		ticket = models.Ticket{
			SeatID:     seat.ID,
			EventID:    eventID,
			UserID:     userID,
			TicketRef:  ticketRef,
			PaidAmount: amount,
		}
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if ae, ok := err.(*apperrors.AppError); ok {
			return nil, ae
		}
		return nil, apperrors.New(http.StatusInternalServerError, "Payment transaction failed", err)
	}

	if err := p.DB.Preload("Event").Preload("Seat").First(&ticket, ticket.ID).Error; err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Failed to load ticket", err)
	}
	return &ticket, nil
}


func (p *PostgresDB) BulkCreateSeatStatuses(statuses []models.SeatStatus) *apperrors.AppError {
	if err := p.DB.CreateInBatches(&statuses, 100).Error; err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to bulk create seat statuses", err)
	}
	return nil
}

func (p *PostgresDB) GetUserTickets(userID uint) ([]models.Ticket, *apperrors.AppError) {
	var tickets []models.Ticket

	if err := p.DB.Preload("Event").Preload("Seat").Where("user_id = ?", userID).Order("created_at DESC").Find(&tickets).Error; err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Failed to retrieve user tickets", err)
	}
	return tickets, nil
}

func (p *PostgresDB) GetEventSeatsWithStatus(eventID uint) ([]models.SeatStatus, *apperrors.AppError) {
	var statuses []models.SeatStatus

	if err := p.DB.Preload("Seat").Joins("JOIN seats ON seats.id = seat_statuses.seat_id").Where("seat_statuses.event_id = ?", eventID).Order("seats.row_name ASC, (split_part(seats.seat_number, '-', 2))::int ASC").Find(&statuses).Error; err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Failed to fetch seat statuses", err)
	}
	return statuses, nil
}

func (p *PostgresDB) GetTicketByRef(ref string) (models.Ticket, *apperrors.AppError) {
	var ticket models.Ticket
	if err := p.DB.Preload("Event").Preload("Seat").Where("ticket_ref = ?", ref).First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return models.Ticket{}, apperrors.New(http.StatusNotFound, "Ticket not found", err)
		}
		return models.Ticket{}, apperrors.New(http.StatusInternalServerError, "Failed to query ticket", err)
	}
	return ticket, nil

}

func (p *PostgresDB) GetAvailableSeatCount(eventID uint) (int64, *apperrors.AppError) {
	var count int64
	if err := p.DB.Model(&models.SeatStatus{}).Where("event_id = ? AND status = ?", eventID, "available").Count(&count).Error; err != nil {
		return 0, apperrors.New(http.StatusInternalServerError, "Failed to count available seats", err)
	}

	return count, nil
}
