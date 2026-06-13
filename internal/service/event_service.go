package service

import (
	"log"
	"net/http"
	"sync"
	"time"

	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/repository"
)

type EventService struct {
	repo *repository.PostgresDB
	wg   *sync.WaitGroup // 👈 تزریق پوینتر WaitGroup برای مدیریت فرآیندهای پس‌زمینه
}

func NewEventService(repo *repository.PostgresDB, wg *sync.WaitGroup) *EventService {
	return &EventService{repo: repo, wg: wg}
}

type CreateEventDTO struct {
	Title       string
	Description string
	Location    string
	StartTime   time.Time
	Rows        int
	SeatsPerRow int
}

func (s *EventService) CreateNewEvent(dto CreateEventDTO) (*models.Event, *apperrors.AppError) {

	event := &models.Event{
		Title:       dto.Title,
		Description: dto.Description,
		Location:    dto.Location,
		StartTime:   dto.StartTime,
		TotalSeats:  dto.Rows * dto.SeatsPerRow,
		Status:    "creating_seats",
	}

	if err := s.repo.CreateEvent(event); err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Internal server error", err)
	}

	s.wg.Add(1)
	go func(id uint, r, sNum int) {
		defer s.wg.Done()

		err := s.repo.CreateSeatsForEvent(id, r, sNum)
        if err != nil {
            log.Printf("[CRITICAL] Failed to generate seats for event %d: %v", id, err)
            s.repo.UpdateEventStatus(id, "failed") // change status to fail
            return
        }

        // seats is ready
        s.repo.UpdateEventStatus(id, "active")
        log.Printf("[ASYNC SUCCESS] Event %d is now active with all seats generated", id)

	}(event.ID, dto.Rows, dto.SeatsPerRow)

	return event, nil

}

func (s *EventService) ListAllEvents() ([]models.Event, *apperrors.AppError) {
	events, err := s.repo.GetAll()
	if err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Error retrieving event list.", err)
	}
	return events, nil
}
