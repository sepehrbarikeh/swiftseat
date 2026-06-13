package repository


import (
	"fmt"

	"swift-seat/internal/models"
	"gorm.io/gorm"
)
func (r *PostgresDB) CreateEventWithSeats(event *models.Event, rows int, seatsPerRow int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(event).Error; err != nil {
			return err
		}

		var seats []models.Seat
		for r := 0; r < rows; r++ {
			rowName := string(rune('A' + r))
			for s := 1; s <= seatsPerRow; s++ {
				seats = append(seats, models.Seat{
					EventID:    event.ID,
					SeatNumber: fmt.Sprintf("%s-%d", rowName, s),
					RowName:    rowName,
					Price:      500000.0,
				})
			}
		}

		if err := tx.Create(&seats).Error; err != nil {
			return err
		}

		var statuses []models.SeatStatus
		for _, seat := range seats {
			statuses = append(statuses, models.SeatStatus{
				SeatID:  seat.ID,
				EventID: event.ID,
				Status:  "available",
			})
		}

		if err := tx.Create(&statuses).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *PostgresDB) GetAll() ([]models.Event, error) {
	var events []models.Event
	err := r.DB.Model(&models.Event{}).Find(&events).Error
	return events, err
}