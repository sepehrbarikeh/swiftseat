package service

import (
	"net/http"
	"time"

	"swift-seat/internal/apperrors"
	"swift-seat/internal/models"
	"swift-seat/internal/repository"
)

type EventService struct {
	repo *repository.PostgresDB
}

func NewEventService(repo *repository.PostgresDB) *EventService {
	return &EventService{repo: repo}
}

type CreateEventDTO struct {
	Title       string
	Description string
	Location    string
	StartTime   string
	Rows        int
	SeatsPerRow int
}

func (s *EventService) CreateNewEvent(dto CreateEventDTO) (*models.Event, *apperrors.AppError) {
	// ۱. ولیدیشن منطق بیزینس (Business Rules Validation)
	if dto.Title == "" || dto.Location == "" {
		return nil, apperrors.NewValidationError("عنوان رویداد و مکان آن نمی‌تواند خالی باشد")
	}
	if dto.Rows <= 0 || dto.SeatsPerRow <= 0 {
		return nil, apperrors.NewValidationError("تعداد ردیف‌ها و صندلی‌ها باید بزرگتر از صفر باشد")
	}

	parsedTime, err := time.Parse(time.RFC3339, dto.StartTime)
	if err != nil {
		return nil, apperrors.New(http.StatusBadRequest, "فرمت تاریخ وارد شده نامعتبر است (RFC3339 مورد نیاز است)", err)
	}

	if parsedTime.Before(time.Now()) {
		return nil, apperrors.NewValidationError("زمان برگزاری رویداد نمی‌تواند در گذشته باشد")
	}

	// ۲. مپ کردن به مدل دیتابیس
	event := &models.Event{
		Title:       dto.Title,
		Description: dto.Description,
		Location:    dto.Location,
		StartTime:   parsedTime,
		TotalSeats:  dto.Rows * dto.SeatsPerRow,
	}

	// ۳. صدا زدن ریپازیتوری
	err = s.repo.CreateEventWithSeats(event, dto.Rows, dto.SeatsPerRow)
	if err != nil {
		// تبدیل خطای خام سیستم به خطای ساختاریافته غنی با کد 500
		return nil, apperrors.New(http.StatusInternalServerError, "خطای غیرمنتظره در سیستم هنگام ذخیره‌سازی رویداد", err)
	}

	return event, nil
}

func (s *EventService) ListAllEvents() ([]models.Event, *apperrors.AppError) {
	events, err := s.repo.GetAll()
	if err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "خطا در دریافت لیست رویدادها", err)
	}
	return events, nil
}