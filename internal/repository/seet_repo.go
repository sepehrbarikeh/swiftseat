package repository

import (
	"errors"

	"time"

	"swift-seat/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (p *PostgresDB) ReserveSeatWithLock(seatNumber string, eventID uint, userID uint, duration time.Duration) error {
	return p.DB.Transaction(func(tx *gorm.DB) error {
		var status models.SeatStatus

		// ۱. پیدا کردن اطلاعات و آیدی اصلی صندلی
		var seat models.Seat
		if err := tx.Where("event_id = ? AND seat_number = ?", eventID, seatNumber).First(&seat).Error; err != nil {
			return errors.New("gorm: record not found")
		}

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id = ? AND event_id = ?", seat.ID, eventID).
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

func (p *PostgresDB) ExecutePaymentTransaction(seatNumber string, eventID, userID uint, amount int64, ticketRef string) (*models.Ticket, error) {
	var ticket models.Ticket

	err := p.DB.Transaction(func(tx *gorm.DB) error {
		var status models.SeatStatus

		var seat models.Seat
		if err := tx.Where("event_id = ? AND seat_number = ?", eventID, seatNumber).First(&seat).Error; err != nil {
			return errors.New("seat_not_found")
		}

		// ۱. قفل کردن سطر صندلی برای جلوگیری از Race Condition
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("seat_id = ? AND event_id = ?", seat.ID, eventID).
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

// BulkCreateSeatStatuses صندلی‌های یک ایونت را به صورت گروهی وارد دیتابیس می‌کند
func (p *PostgresDB) BulkCreateSeatStatuses(statuses []models.SeatStatus) error {
	// GORM به صورت خودکار با متد Create روی یک اسلایس (آرایه)، Bulk Insert می‌زند.
	// پارامتر دوم (مثلاً 100) می‌گوید دیتاها را در دسته‌های 100 تایی دسته‌بندی و ایمپورت کن.
	return p.DB.CreateInBatches(&statuses, 100).Error
}

func (p *PostgresDB) GetUserTickets(userID uint) ([]models.Ticket, error) {
	var tickets []models.Ticket

	err := p.DB.
		Preload("Event").
		Preload("Seat").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tickets).Error

	if err != nil {
		return nil, err
	}
	return tickets, nil
}

func (p *PostgresDB) GetEventSeatsWithStatus(eventID uint) ([]models.SeatStatus, error) {
	var statuses []models.SeatStatus

	err := p.DB.
		Preload("Seat").
		Where("event_id = ?", eventID).
		Find(&statuses).Error

	if err != nil {
		return nil, err
	}
	return statuses, nil
}

func (p *PostgresDB) GetPaginatedEvents(page, limit int, search, location string) ([]models.Event, int64, error) {
	var events []models.Event
	var total int64

	// ایجاد یک کوئری پایه روی مدل ایونت
	query := p.DB.Model(&models.Event{})

	// ۱. اعمال سرچ اختیاری روی عنوان (Case-Insensitive)
	if search != "" {
		searchTerm := "%" + search + "%"
		// سرچ هم‌زمان روی عنوان یا لوکیشن ایونت
		query = query.Where("title ILIKE ? OR location ILIKE ?", searchTerm, searchTerm)
	}

	// ۲. اعمال فیلتر اختیاری روی مکان
	if location != "" {
		query = query.Where("location = ?", location)
	}

	// ۳. گرفتن تعداد کل رکوردها با فیلترهای اعمال شده (برای متادیتای فرانت‌آند)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ۴. اعمال صفحه‌بندی و دریافت دیتا
	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (p *PostgresDB) GetTicketByRef(ref string) (*models.Ticket, error) {
	var ticket models.Ticket
	// استفاده از Preload برای گرفتن دیتایِ ایونت و صندلی همراه با تیکت
	err := p.DB.Preload("Event").Preload("Seat").Where("ticket_ref = ?", ref).First(&ticket).Error
	return &ticket, err

}
