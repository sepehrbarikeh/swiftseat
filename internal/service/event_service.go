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

func (s *EventService) UpdateEvent(c *fiber.Ctx, id string, fileHeader *multipart.FileHeader, dto CreateEventDTO) *apperrors.AppError {
	ctx := context.Background()
	// ۱. گرفتن ایونت قدیمی
	oldEvent, err := s.repo.FindByID(id)
	if err != nil {
		return apperrors.New(http.StatusNotFound, "Event not found", err)
	}

	// ۲. آپدیت کردن فیلدها
	oldEvent.Title = dto.Title
	oldEvent.Description = dto.Description
	oldEvent.Location = dto.Location
	oldEvent.StartTime = dto.StartTime

	// اگر فایل جدیدی هست، آپلود کن و آدرس جدید رو جایگزین کن
	if fileHeader != nil {
		// پاک کردن فایل قدیمی (بعد از اطمینان از موفقیت آپدیت)
		_ = os.Remove(oldEvent.ImageURL)

		err = c.SaveFile(fileHeader, dto.ImageUrl)
		if err != nil {
			return apperrors.New(500, "Failed to save new image", err)
		}
		oldEvent.ImageURL = dto.ImageUrl
	}

	// ۳. ذخیره در دیتابیس (فراخوانی متد اصلاح شده ریپو)
	if err := s.repo.UpdateEvent(oldEvent); err != nil {
		return apperrors.New(500, "Update failed", err)
	}
	if err := s.redis.DeleteCache(ctx, "events:active"); err != nil {
		log.Printf("[WARN] Failed to invalidate Redis cache after event activation: %v", err)
	}

	return nil
}

func (s *EventService) DeleteEvent(id string) *apperrors.AppError {
	ctx := context.Background()
	lastFile, err := s.repo.FindByID(id)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "Error retrieving event list.", err)
	}
	err = s.repo.DeleteEvent(id)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "Error retrieving event list.", err)
	}
	err = os.Remove(lastFile.ImageURL)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "Error retrieving event list.", err)
	}
	if err := s.redis.DeleteCache(ctx, "events:active"); err != nil {
		log.Printf("[WARN] Failed to invalidate Redis cache after event activation: %v", err)
	}
	return nil
}

// public
func (s *EventService) GetPublicEvents(ctx context.Context, page, limit int, search, location string) (*PaginatedEventsResponse, error) {
   
    cacheKey := fmt.Sprintf("events:public:p%d:l%d:s:%s:l:%s", page, limit, search, location)

    var response PaginatedEventsResponse
    found, _ := s.redis.GetCache(ctx, cacheKey, &response)
    if found {
        return &response, nil
    }

    events, totalItems, err := s.repo.GetEventsPaginated(page, limit, search, location, "active")
    if err != nil {
        return nil, err
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
    // ارسال رشته خالی برای status (همه را می‌آورد)
    events, totalItems, err := s.repo.GetEventsPaginated(page, limit, search, location, "")
    if err != nil {
        return nil, apperrors.New(http.StatusInternalServerError, "Failed to retrieve events", err)
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

// internal/service/event_service.go

func (s *EventService) GetHomeEvents() (map[string][]EventResponse, error) {
    // گرفتن لیست‌ها
    popular, _ := s.repo.GetPopularEvents(4)
    upcoming, _ := s.repo.GetUpcomingEvents(4)

    // تابع کمکی برای ترکیب با صندلی‌ها
    getWithSeats := func(events []models.Event) []EventResponse {
        var ids []uint
        for _, e := range events { ids = append(ids, e.ID) }
        counts, _ := s.repo.GetAvailableSeatCounts(ids) // همان متدِ دسته‌ای قبلی

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