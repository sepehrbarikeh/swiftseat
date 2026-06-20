package repository

import (
	"net/http"
	"time"

	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/pkg/ticket"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (p *PostgresDB) CreateReservation(seatNumbers []string, eventID uint, userID uint, duration time.Duration) (*models.Reservation, *apperrors.AppError) {
	var reservation models.Reservation
	ticketRef := ticket.GenerateTicketRef()

	err := p.DB.Transaction(func(tx *gorm.DB) error {
		expiresAt := time.Now().Add(duration)

		// 1. ایجاد رزرو
		reservation = models.Reservation{
			Ref:         ticketRef,
			UserID:      userID,
			EventID:     eventID,
			Status:      models.ReservationReserved,
			ExpiresAt:   &expiresAt,
			TotalAmount: 0,
		}

		if err := tx.Create(&reservation).Error; err != nil {
			return err
		}

		// 2. قفل کردن صندلی‌ها جهت بررسی وضعیت
		var seats []models.Seat
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("event_id = ? AND seat_number IN ?", eventID, seatNumbers).
			Find(&seats).Error; err != nil {
			return err
		}

		if len(seats) != len(seatNumbers) {
			return apperrors.New(http.StatusNotFound, "برخی صندلی‌ها یافت نشدند", nil)
		}

		// 3. ایجاد رکوردهای ReservationSeat
		var rs []models.ReservationSeat
		for _, seat := range seats {
			rs = append(rs, models.ReservationSeat{
				ReservationID: reservation.ID,
				SeatID:        seat.ID,
			})
		}
		if err := tx.Create(&rs).Error; err != nil {
			return err
		}

		// 4. دریافت و آپدیت وضعیت صندلی‌ها
		seatIDs := make([]uint, len(seats))
		for i, s := range seats {
			seatIDs[i] = s.ID
		}

		var statuses []models.SeatStatus
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id IN ? AND event_id = ?", seatIDs, eventID).
			Find(&statuses).Error; err != nil {
			return err
		}

		for _, st := range statuses {
			// اعتبارسنجی وضعیت
			if st.Status == "sold" {
				return apperrors.New(http.StatusConflict, "صندلی فروخته شده است", nil)
			}
			if st.Status == "reserved" && st.ExpiresAt != nil && st.ExpiresAt.After(time.Now()) {
				return apperrors.New(http.StatusConflict, "صندلی قبلاً رزرو شده است", nil)
			}

			// آپدیت تکی هر صندلی (جایگزین Bulk Save برای رفع خطای 42P10)
			err := tx.Model(&st).Updates(map[string]interface{}{
				"status":      "reserved",
				"reserved_by": userID,
				"expires_at":  reservation.ExpiresAt,
			}).Error

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if ae, ok := err.(*apperrors.AppError); ok {
			return nil, ae
		}
		return nil, apperrors.New(http.StatusInternalServerError, "خطا در انجام رزرو", err)
	}

	return &reservation, nil
}

func (p *PostgresDB) ExecutePaymentTransaction(reservationRef string, userID uint, amount int64) (*models.Ticket, *apperrors.AppError) {

	var lastTicket models.Ticket

	err := p.DB.Transaction(func(tx *gorm.DB) error {

		now := time.Now()

		// 1. get reservation
		var reservation models.Reservation

		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("ref = ?", reservationRef).
			First(&reservation).Error; err != nil {

			return apperrors.New(http.StatusNotFound, "Reservation not found", err)
		}

		// 2. validate
		if reservation.UserID != userID {
			return apperrors.New(http.StatusForbidden, "Invalid user", nil)
		}

		if reservation.ExpiresAt == nil ||
			reservation.ExpiresAt.Before(now) {

			return apperrors.New(http.StatusGone, "Expired", nil)
		}

		if reservation.Status == models.ReservationPaid {
			return apperrors.New(http.StatusConflict, "Already sold", nil)
		}

		// 3. mark paid
		reservation.Status = "paid"

		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}

		// 4. get seats
		var seats []models.Seat

		if err := tx.
			Table("seats").
			Joins("JOIN reservation_seats rs ON rs.seat_id = seats.id").
			Where("rs.reservation_id = ?", reservation.ID).
			Find(&seats).Error; err != nil {

			return err
		}

		if err := tx.Model(&models.SeatStatus{}).
			Where("event_id = ? AND seat_id IN (?)",
				reservation.EventID,
				tx.Table("reservation_seats").
					Select("seat_id").
					Where("reservation_id = ?", reservation.ID),
			).
			Updates(map[string]interface{}{
				"status":      "sold",
				"reserved_by": nil,
				"expires_at":  nil,
			}).Error; err != nil {

			return err
		}

		// 5. create tickets
		tickets := make([]models.Ticket, 0, len(seats))

		for _, seat := range seats {
			tickets = append(tickets, models.Ticket{
				SeatID:     seat.ID,
				EventID:    reservation.EventID,
				UserID:     userID,
				TicketRef:  reservation.Ref,
				PaidAmount: amount,
			})
		}

		if len(tickets) > 0 {
			if err := tx.Create(&tickets).Error; err != nil {
				return err
			}
			lastTicket = tickets[len(tickets)-1]
		}

		return nil
	})

	if err != nil {
		if ae, ok := err.(*apperrors.AppError); ok {
			return nil, ae
		}

		return nil, apperrors.New(http.StatusInternalServerError, "Payment failed", err)
	}

	return &lastTicket, nil
}

// CleanupExpiredSeats
func (p *PostgresDB) CleanupExpiredSeats() (int64, *apperrors.AppError) {

	now := time.Now()

	result := p.DB.Model(&models.SeatStatus{}).
		Where(`
			status = ? 
			AND expires_at IS NOT NULL 
			AND expires_at < ?
		`, "reserved", now).
		Updates(map[string]interface{}{
			"status":      "available",
			"reserved_by": nil,
			"expires_at":  nil,
		})

	if result.Error != nil {
		return 0, apperrors.New(
			http.StatusInternalServerError,
			"Failed to cleanup expired seats",
			result.Error,
		)
	}

	return result.RowsAffected, nil
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
