package service

import (
	"net/http"
	"time"

	"swift-seat/internal/pkg/apperrors"
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
	if dto.Title == "" || dto.Location == "" {
		return nil, apperrors.NewValidationError("Event title and location cannot be empty")
	}
	if dto.Rows <= 0 || dto.SeatsPerRow <= 0 {
		return nil, apperrors.NewValidationError("Number of rows and seats must be greater than zero")
	}

	parsedTime, err := time.Parse(time.RFC3339, dto.StartTime)
	if err != nil {
		return nil, apperrors.New(http.StatusBadRequest, "Invalid date format", err)
	}

	if parsedTime.Before(time.Now()) {
		return nil, apperrors.NewValidationError("The time of the event cannot be in the past.")
	}

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
		return nil, apperrors.New(http.StatusInternalServerError, "Unexpected error", err)
	}

	return event, nil
}

func (s *EventService) ListAllEvents() ([]models.Event, *apperrors.AppError) {
	events, err := s.repo.GetAll()
	if err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Error retrieving event list.", err)
	}
	return events, nil
}