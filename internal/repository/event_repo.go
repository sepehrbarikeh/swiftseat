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

func (p *PostgresDB) BulkCreateSeats(seats []models.SeatStatus) error {
	if len(seats) == 0 {
		return nil
	}
	return p.DB.CreateInBatches(&seats, 200).Error
}

func (p *PostgresDB) UpdateEventStatus(eventID uint, status string) error {
	return p.DB.Model(&models.Event{}).Where("id = ?", eventID).Update("status", status).Error
}

func (p *PostgresDB) FindByID(id string) (models.Event, error) {
	var event models.Event

	err := p.DB.Where("id = ?", id).First(&event, id).Error
	if err != nil {
		return models.Event{},err
	}
	return event, nil
}

func (p *PostgresDB) UpdateEvent(event models.Event) error {
	return p.DB.Save(event).Error
}

func (p *PostgresDB) DeleteEvent(id string) error {
	return p.DB.Delete(&models.Event{}, id).Error
}

func (p *PostgresDB) GetEventsPaginated(page, limit int, search, location string, statusFilter string) ([]models.Event, int64, error) {
    // ایجاد پایه کوئری
    query := p.DB.Model(&models.Event{})

    // ۱. فیلتر وضعیت (اگر خالی باشد، همه وضعیت‌ها برمی‌گردند - مناسب برای ادمین)
    if statusFilter != "" {
        query = query.Where("status = ?", statusFilter)
    }

    // ۲. فیلتر جستجو (هم در عنوان هم در توضیحات)
    if search != "" {
        searchPattern := "%" + search + "%"
        query = query.Where("title LIKE ? OR description LIKE ?", searchPattern, searchPattern)
    }

    // ۳. فیلتر مکان
    if location != "" {
        query = query.Where("location = ?", location)
    }

    // ۴. دریافت تعداد کل (قبل از اعمال Limit و Offset برای محاسبه صفحات)
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // ۵. دریافت دیتا با اعمال Pagination
    var events []models.Event
    offset := (page - 1) * limit
    err := query.Offset(offset).Limit(limit).Order("start_time asc").Find(&events).Error
    
    return events, total, err
}