package repository

import (
	"fmt"
	"net/http"
	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
	"time"

	"gorm.io/gorm"
)

type SeatCountResult struct {
	EventID uint
	Count   int64
}

// CreateEvent فقط خود ایونت را می‌سازد (کاملاً Sync)
func (p *PostgresDB) CreateEvent(event *models.Event) *apperrors.AppError {
	if err := p.DB.Create(event).Error; err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to create event", err)
	}
	return nil
}

// CreateSeatsForEvent صندلی‌ها و وضعیت‌ها را می‌سازد (می‌تواند Async صدا زده شود)
func (p *PostgresDB) CreateSeatsForEvent(eventID uint, rows int, seatsPerRow int) *apperrors.AppError {
	if err := p.DB.Transaction(func(tx *gorm.DB) error {
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
	}); err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to create seats for event", err)
	}
	return nil
}

func (p *PostgresDB) BulkCreateSeats(seats []models.SeatStatus) *apperrors.AppError {
	if len(seats) == 0 {
		return nil
	}
	if err := p.DB.CreateInBatches(&seats, 200).Error; err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to bulk create seats", err)
	}
	return nil
}

func (p *PostgresDB) UpdateEventStatus(eventID uint, status string) *apperrors.AppError {
	if err := p.DB.Model(&models.Event{}).Where("id = ?", eventID).Update("status", status).Error; err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to update event status", err)
	}
	return nil
}

func (p *PostgresDB) FindByID(id string) (models.Event, *apperrors.AppError) {
	var event models.Event

	err := p.DB.Where("id = ?", id).First(&event, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return models.Event{}, apperrors.New(http.StatusNotFound, "Event not found", err)
		}
		return models.Event{}, apperrors.New(http.StatusInternalServerError, "Failed to fetch event", err)
	}
	return event, nil
}

func (p *PostgresDB) UpdateEvent(event models.Event) *apperrors.AppError {
	if err := p.DB.Save(event).Error; err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to update event", err)
	}
	return nil
}

func (p *PostgresDB) DeleteEvent(id string) *apperrors.AppError {
	if err := p.DB.Delete(&models.Event{}, id).Error; err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to delete event", err)
	}
	return nil
}

func (p *PostgresDB) GetEventsPaginated(page, limit int, search, location string, statusFilter string) ([]models.Event, int64, *apperrors.AppError) {
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
		return nil, 0, apperrors.New(http.StatusInternalServerError, "Failed to count events", err)
	}

	// ۵. دریافت دیتا با اعمال Pagination
	var events []models.Event
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("start_time asc").Find(&events).Error; err != nil {
		return nil, 0, apperrors.New(http.StatusInternalServerError, "Failed to fetch events", err)
	}

	return events, total, nil
}

func (p *PostgresDB) GetAvailableSeatCounts(eventIDs []uint) (map[uint]int64, *apperrors.AppError) {
	var results []SeatCountResult

	// کوئری برای گرفتن تعداد صندلی‌های available به صورت گروهی
	if err := p.DB.Model(&models.SeatStatus{}).
		Select("event_id, count(*) as count").
		Where("event_id IN ? AND status = ?", eventIDs, "available").
		Group("event_id").
		Scan(&results).Error; err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Failed to get available seat counts", err)
	}

	// تبدیل به مپ برای دسترسی سریع (O(1))
	counts := make(map[uint]int64)
	for _, res := range results {
		counts[res.EventID] = res.Count
	}
	return counts, nil
}

// GetPopularEvents ایونت‌هایی با بیشترین صندلی فروخته شده
func (p *PostgresDB) GetPopularEvents(limit int) ([]models.Event, *apperrors.AppError) {
	var events []models.Event
	if err := p.DB.Model(&models.Event{}).
		Joins("JOIN seat_statuses ON seat_statuses.event_id = events.id").
		Where("seat_statuses.status = ?", "booked").
		Group("events.id").
		Order("count(seat_statuses.id) DESC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Failed to fetch popular events", err)
	}
	return events, nil
}

// GetUpcomingEvents ایونت‌هایی که زمان شروع‌شان نزدیک‌تر است
func (p *PostgresDB) GetUpcomingEvents(limit int) ([]models.Event, *apperrors.AppError) {
	var events []models.Event
	// فقط ایونت‌های اکتیو و آینده
	if err := p.DB.Where("status = ? AND start_time > ?", "active", time.Now()).
		Order("start_time ASC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Failed to fetch upcoming events", err)
	}
	return events, nil
}
