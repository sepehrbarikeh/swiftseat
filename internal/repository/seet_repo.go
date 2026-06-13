package repository

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
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

func (p *PostgresDB) ConfirmPayment(seatID, eventID, userID uint, amount int64) error {
	err := p.DB.Transaction(func(tx *gorm.DB) error {
		var status models.SeatStatus

		// ۱. پیدا کردن صندلی و قفل کردن آن برای این تراکنش
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id = ? AND event_id = ?", seatID, eventID).
			First(&status).Error
		if err != nil {
			return fmt.Errorf("seat_not_found")
		}

		// ۲. اعتبارسنجی وضعیت صندلی
		if status.Status != "reserved" || status.ReservedBy == nil || *status.ReservedBy != userID {
			return fmt.Errorf("not_your_reservation")
		}

		if status.ExpiresAt != nil && status.ExpiresAt.Before(time.Now()) {
			return fmt.Errorf("reservation_expired")
		}

		// ۳. تغییر وضعیت صندلی به فروخته شده
		status.Status = "sold"
		status.ExpiresAt = nil // ددلاین پاک می‌شود چون خرید قطعی شده
		if err := tx.Save(&status).Error; err != nil {
			return err
		}

		// ۴. صدور بلیت نهایی و تولید کد پیگیری رندوم
		ticketRef := generateTicketRef()
		ticket := models.Ticket{
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
	return err
}

// تابع کمکی برای تولید کد پیگیری امن و رندوم
func generateTicketRef() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("TIC-%d", n.Int64()+100000) // نمونه: TIC-548329
}