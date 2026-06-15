package service

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"

	"swift-seat/internal/database"
	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type EventService struct {
	repo  *repository.PostgresDB
	redis *database.RedisClient
	wg    *sync.WaitGroup // 👈 تزریق پوینتر WaitGroup برای مدیریت فرآیندهای پس‌زمینه
}

type EventResponse struct {
	models.Event
	AvailableSeats int64 `json:"available_seats"`
	IsSoldOut      bool  `json:"is_sold_out"`
}

type UpdateEventRequest struct {
	Title       string                `form:"title"` // فقط اطلاعاتی که کاربر مجاز است ویرایش کند
	Description string                `form:"description"`
	Location    string                `form:"location"`
	Image       *multipart.FileHeader `form:"image"` // فایلی که می‌خواهیم آپلود کنیم
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

func (s *EventService) CreateNewEvent(c *fiber.Ctx, fileHeader *multipart.FileHeader, dto CreateEventDTO) (*models.Event, *apperrors.AppError) {

	if err := c.SaveFile(fileHeader, dto.ImageUrl); err != nil {
		return nil, apperrors.New(500, "Internal Error", err)
	}

	event := &models.Event{
		Title:       dto.Title,
		Description: dto.Description,
		Location:    dto.Location,
		StartTime:   dto.StartTime,
		TotalSeats:  dto.Rows * dto.SeatsPerRow,
		Status:      "creating_seats",
		ImageURL:    dto.ImageUrl,
	}

	if appErr := s.repo.CreateEvent(event); appErr != nil {
		return nil, appErr
	}

	s.wg.Add(1)
	go func(id uint, r, sNum int) {
		defer s.wg.Done()

		ctx := context.Background()

		if appErr := s.repo.CreateSeatsForEvent(id, r, sNum); appErr != nil {
			log.Printf("[CRITICAL] Failed to generate seats for event %d: %v", id, appErr)
			_ = s.repo.UpdateEventStatus(id, "failed") // change status to fail
			return
		}

		// seats is ready
		_ = s.repo.UpdateEventStatus(id, "active")

		// ۲. 🚀 باطل کردن کش: چون یک ایونت اکتیو جدید داریم، کش قبلی ردیس را پاک می‌کنیم
		if err := s.redis.DeleteCache(ctx, "events:active"); err != nil {
			log.Printf("[WARN] Failed to invalidate Redis cache after event activation: %v", err)
		}

		log.Printf("[ASYNC SUCCESS] Event %d is active and cache invalidated", id)

	}(event.ID, dto.Rows, dto.SeatsPerRow)

	return event, nil

}

func (s *EventService) UpdateEvent(c *fiber.Ctx, id string, fileHeader *multipart.FileHeader, dto CreateEventDTO) *apperrors.AppError {
	ctx := context.Background()
	
	oldEvent, appErr := s.repo.FindByID(id)
	if appErr != nil {
		return appErr
	}

	// ۲. آپدیت کردن فیلدها
	oldEvent.Title = dto.Title
	oldEvent.Description = dto.Description
	oldEvent.Location = dto.Location
	oldEvent.StartTime = dto.StartTime

	
	if fileHeader != nil {
		
		_ = os.Remove(oldEvent.ImageURL)

		if err := c.SaveFile(fileHeader, dto.ImageUrl); err != nil {
			return apperrors.New(500, "Failed to save new image", err)
		}
		oldEvent.ImageURL = dto.ImageUrl
	}

	
	if appErr := s.repo.UpdateEvent(oldEvent); appErr != nil {
		return appErr
	}
	if err := s.redis.DeleteCache(ctx, "events:active"); err != nil {
		log.Printf("[WARN] Failed to invalidate Redis cache after event activation: %v", err)
	}

	return nil
}

func (s *EventService) DeleteEvent(id string) *apperrors.AppError {
	ctx := context.Background()
	lastFile, appErr := s.repo.FindByID(id)
	if appErr != nil {
		return appErr
	}
	if appErr := s.repo.DeleteEvent(id); appErr != nil {
		return appErr
	}
	if err := os.Remove(lastFile.ImageURL); err != nil {
		return apperrors.New(http.StatusInternalServerError, "Error retrieving event list.", err)
	}
	if err := s.redis.DeleteCache(ctx, "events:active"); err != nil {
		log.Printf("[WARN] Failed to invalidate Redis cache after event activation: %v", err)
	}
	return nil
}

// public
func (s *EventService) GetPublicEvents(ctx context.Context, page, limit int, search, location string) (*PaginatedEventsResponse, *apperrors.AppError) {

	cacheKey := fmt.Sprintf("events:public:p%d:l%d:s:%s:l:%s", page, limit, search, location)

	var response PaginatedEventsResponse
	found, _ := s.redis.GetCache(ctx, cacheKey, &response)
	if found {
		return &response, nil
	}

	events, totalItems, appErr := s.repo.GetEventsPaginated(page, limit, search, location, "active")
	if appErr != nil {
		return nil, appErr
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	response = PaginatedEventsResponse{
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: page,
		Limit:       limit,
		Events:      events,
	}

	_ = s.redis.SetCache(ctx, cacheKey, response, 10*time.Minute)
	return &response, nil
}

// (Protected)
func (s *EventService) GetAdminEvents(page, limit int, search, location string) (*PaginatedEventsResponse, *apperrors.AppError) {
	
	events, totalItems, appErr := s.repo.GetEventsPaginated(page, limit, search, location, "")
	if appErr != nil {
		return nil, appErr
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	return &PaginatedEventsResponse{
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: page,
		Limit:       limit,
		Events:      events,
	}, nil
}



func (s *EventService) GetHomeEvents() (map[string][]EventResponse, *apperrors.AppError) {
	
	popular, appErr := s.repo.GetPopularEvents(4)
	if appErr != nil {
		return nil, appErr
	}
	upcoming, appErr := s.repo.GetUpcomingEvents(4)
	if appErr != nil {
		return nil, appErr
	}

	
	getWithSeats := func(events []models.Event) []EventResponse {
		var ids []uint
		for _, e := range events {
			ids = append(ids, e.ID)
		}
		counts, _ := s.repo.GetAvailableSeatCounts(ids) // ignore minor cache errors

		var res []EventResponse
		for _, e := range events {
			count := counts[e.ID]
			res = append(res, EventResponse{
				Event:          e,
				AvailableSeats: count,
				IsSoldOut:      count == 0,
			})
		}
		return res
	}

	return map[string][]EventResponse{
		"popular":  getWithSeats(popular),
		"upcoming": getWithSeats(upcoming),
	}, nil
}
