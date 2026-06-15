package repository

import (
	"fmt"
	"swift-seat/internal/models"

	"gorm.io/gorm"
)

// internal/repository/postgres.go

// CreateEvent فقط خود ایونت را می‌سازد (کاملاً Sync)
func (p *PostgresDB) CreateEvent(event *models.Event) error {
	return p.DB.Create(event).Error
}

// CreateSeatsForEvent صندلی‌ها و وضعیت‌ها را می‌سازد (می‌تواند Async صدا زده شود)
func (p *PostgresDB) CreateSeatsForEvent(eventID uint, rows int, seatsPerRow int) error {
	return p.DB.Transaction(func(tx *gorm.DB) error {
		var seats []models.Seat
		for r := 0; r < rows; r++ {
			rowName := string(rune('A' + r))
			for s := 1; s <= seatsPerRow; s++ {
				seats = append(seats, models.Seat{
					EventID:    eventID, // آی‌دی واقعی ایونت
					SeatNumber: fmt.Sprintf("%s-%d", rowName, s),
					RowName:    rowName,
					Price:      500000.0,
				})
			}
		}

		if err := tx.CreateInBatches(&seats, 200).Error; err != nil {
			return err
		}

		var statuses []models.SeatStatus
		for _, seat := range seats {
			statuses = append(statuses, models.SeatStatus{
				SeatID:  seat.ID,
				EventID: eventID,
				Status:  "available",
			})
		}

		return tx.CreateInBatches(&statuses, 200).Error
	})
}

func (p *PostgresDB) GetAll() ([]models.Event, error) {
	var events []models.Event
	err := p.DB.Model(&models.Event{}).Find(&events).Error
	return events, err
}

func (p *PostgresDB) BulkCreateSeats(seats []models.SeatStatus) error {
	if len(seats) == 0 {
		return nil
	}
	return p.DB.CreateInBatches(&seats, 200).Error
}

func (p *PostgresDB) UpdateEventStatus(eventID uint, status string) error {
	return p.DB.Model(&models.Event{}).Where("id = ?", eventID).Update("status", status).Error
}

func (p *PostgresDB) GetActiveEvents() ([]models.Event, error) {
	var events []models.Event

	err := p.DB.
		Where("status = ?", "active").
		Order("start_time asc").
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

// func (p *PostgresDB) UpdateEvent(eventID uint, status string) error {

// }

// func (p *PostgresDB) DeletEvent(id uint) error {
// 	return p.DB.Delete(&models.Event{}, id).Error

// 	lastFile, err := s.repo.FindByID(ctx, id, userID)
// 	if err != nil {
// 		return err
// 	}
// 	err = s.repo.DeleteMedia(ctx, id, userID)
// 	if err != nil {
// 		return err
// 	}
// 	err = os.Remove(lastFile.FilePath)
// 	if err != nil {
// 		return errs.Internal(constant.Internal, err)
// 	}
// 	return nil
// }
