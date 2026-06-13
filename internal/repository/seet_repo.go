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