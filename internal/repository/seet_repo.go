package repository

import (
	"errors"
	"time"

	"swift-seat/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (p *PostgresDB) ReserveSeatWithLock(seatID uint, eventID uint, userID uint, duration time.Duration) error {
	return p.DB.Transaction(func(tx *gorm.DB) error {
		var status models.SeatStatus

		// locking for race condition
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id = ? AND event_id = ?", seatID, eventID).
			First(&status).Error

		if err != nil {
			return err
		}

		now := time.Now()
		if status.Status == "sold" {
			return errors.New("seat_already_sold")
		}
		if status.Status == "reserved" && status.ExpiresAt != nil && status.ExpiresAt.After(now) {
			return errors.New("seat_already_reserved")
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
	})
}

// CleanupExpiredSeats
func (p *PostgresDB) CleanupExpiredSeats() (int64, error) {
	now := time.Now()

	result := p.DB.Model(&models.SeatStatus{}).
		Where("status = ? AND expires_at < ?", "reserved", now).
		Updates(map[string]interface{}{
			"status":      "available",
			"reserved_by": nil,
			"expires_at":  nil,
		})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil

}

// فرض می‌کنیم این متد به اینترفیس یا استراکت SeatRepository اضافه میشه
func (p *PostgresDB) ExecutePaymentTransaction(seatID, eventID, userID uint, amount int64, ticketRef string) (*models.Ticket, error) {
	var ticket models.Ticket

	err := p.DB.Transaction(func(tx *gorm.DB) error {
		var status models.SeatStatus

		// ۱. قفل کردن سطر صندلی برای جلوگیری از Race Condition
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id = ? AND event_id = ?", seatID, eventID).
			First(&status).Error
		if err != nil {
			return errors.New("seat_not_found")
		}

		// ۲. اعتبارسنجی وضعیت رزرو صندلی
		if status.Status != "reserved" || status.ReservedBy == nil || *status.ReservedBy != userID {
			return errors.New("not_your_reservation")
		}
		if status.ExpiresAt != nil && status.ExpiresAt.Before(time.Now()) {
			return errors.New("reservation_expired")
		}

		// ۳. قطعی کردن خرید صندلی
		status.Status = "sold"
		status.ExpiresAt = nil
		if err := tx.Save(&status).Error; err != nil {
			return err
		}

		// ۴. ایجاد رکورد بلیت
		ticket = models.Ticket{
			SeatID:     seatID,
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
		return nil, err
	}

	err = p.DB.
		Preload("Event").
		Preload("Seat").
		First(&ticket, ticket.ID).Error

	if err != nil {
		return nil, err
	}

	return &ticket, nil
}

