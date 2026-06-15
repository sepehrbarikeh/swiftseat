package service

import (
	"context"
	"log"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"swift-seat/internal/database"
	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/repository"
)

type EventService struct {
	repo  *repository.PostgresDB
	redis *database.RedisClient
	wg    *sync.WaitGroup // 👈 تزریق پوینتر WaitGroup برای مدیریت فرآیندهای پس‌زمینه
}


type UpdateEventRequest struct {
    Title       string                `form:"title"`       // فقط اطلاعاتی که کاربر مجاز است ویرایش کند
    Description string                `form:"description"`
    Location    string                `form:"location"`
    Image       *multipart.FileHeader `form:"image"`       // فایلی که می‌خواهیم آپلود کنیم
}

func NewEventService(repo *repository.PostgresDB, wg *sync.WaitGroup, redis *database.RedisClient) *EventService {
	return &EventService{
		repo:  repo,
		wg:    wg,
		redis: redis,
	}
}

type CreateEventDTO struct {
	Title       string
	Description string
	Location    string
	StartTime   time.Time
	Rows        int
	SeatsPerRow int
	ImageUrl    string
}

func (s *EventService) GetActiveEvents(ctx context.Context) ([]models.Event, string, error) {
	cacheKey := "events:active"
	var events []models.Event

	found, err := s.redis.GetCache(ctx, cacheKey, &events)
	if err != nil {
		log.Printf("[WARN] Redis failed in Service layer: %v", err)
	}
	if found {

		return events, "cache", nil
	}

	events, err = s.repo.GetActiveEvents()
	if err != nil {
		return nil, "", err
	}

	// ۳. اگر دیتایی بود، برای دفعات بعدی در ردیس ذخیره می‌کنیم (انقضا: ۱۰ دقیقه)
	if len(events) > 0 {
		if err := s.redis.SetCache(ctx, cacheKey, events, 10*time.Minute); err != nil {
			log.Printf("[WARN] Failed to populate Redis cache: %v", err)
		}
	}

	return events, "database", nil
}

func (s *EventService) CreateNewEvent(dto CreateEventDTO) (*models.Event, *apperrors.AppError) {

	event := &models.Event{
		Title:       dto.Title,
		Description: dto.Description,
		Location:    dto.Location,
		StartTime:   dto.StartTime,
		TotalSeats:  dto.Rows * dto.SeatsPerRow,
		Status:      "creating_seats",
		ImageURL:    dto.ImageUrl,
	}

	if err := s.repo.CreateEvent(event); err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "Internal server error", err)
	}

	s.wg.Add(1)
	go func(id uint, r, sNum int) {
		defer s.wg.Done()

		ctx := context.Background()

		err := s.repo.CreateSeatsForEvent(id, r, sNum)
		if err != nil {
			log.Printf("[CRITICAL] Failed to generate seats for event %d: %v", id, err)
			s.repo.UpdateEventStatus(id, "failed") // change status to fail
			return
		}

		// seats is ready
		s.repo.UpdateEventStatus(id, "active")

		// ۲. 🚀 باطل کردن کش: چون یک ایونت اکتیو جدید داریم، کش قبلی ردیس را پاک می‌کنیم
		if err := s.redis.DeleteCache(ctx, "events:active"); err != nil {
			log.Printf("[WARN] Failed to invalidate Redis cache after event activation: %v", err)
		}

		log.Printf("[ASYNC SUCCESS] Event %d is active and cache invalidated", id)

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
