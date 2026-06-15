package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"swift-seat/internal/service"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CreateEventRequest struct {
	Title       string                `form:"title"`
	Description string                `form:"description"`
	Location    string                `form:"location"`
	StartTime   string                `form:"start_time"`
	Rows        int                   `form:"rows"`
	SeatsPerRow int                   `form:"seats_per_row"`
	Image       *multipart.FileHeader `form:"image"` // برای دریافت فایل
}

type UpdateEventRequest struct {
	Title       string                `form:"title"` // فقط اطلاعاتی که کاربر مجاز است ویرایش کند
	Description string                `form:"description"`
	Location    string                `form:"location"`
	Image       *multipart.FileHeader `form:"image"` // فایلی که می‌خواهیم آپلود کنیم
}

// @Summary Create an event
// @Description Create a new event in the system
// @Tags Events
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param event body CreateEventRequest true "Event payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/events [post]
func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	title := c.FormValue("title")
	description := c.FormValue("description")
	location := c.FormValue("location")
	startTimeStr := c.FormValue("start_time")
	rows, _ := strconv.Atoi(c.FormValue("rows"))
	seatsPerRow, _ := strconv.Atoi(c.FormValue("seats_per_row"))

	if title == "" || location == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Event title and location required"})
	}
	if rows <= 0 || seatsPerRow <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid seat dimensions"})
	}

	parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date format"})
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		if !strings.HasPrefix(fileHeader.Header.Get("Content-Type"), "image/") {
			return fiber.NewError(fiber.StatusBadRequest, "Bad Request")
		}
	}
	fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileHeader.Filename)
	savePath := fmt.Sprintf("./uploads/%s", fileName)

	dto := service.CreateEventDTO{
		Title:       title,
		Description: description,
		Location:    location,
		StartTime:   parsedTime,
		Rows:        rows,
		SeatsPerRow: seatsPerRow,
		ImageUrl:    savePath,
	}

	event, appErr := h.svc.CreateNewEvent(c, fileHeader, dto)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message":  "event created",
		"event_id": event.ID,
	})
}

// ۲. نسخه ساده‌تر و بهینه برای آپدیت (با هندل کردنِ اختیاریِ فایل)
func (h *EventHandler) UpdateEvent(c *fiber.Ctx) error {
	id := c.Params("id")

	// دریافت فایل (اختیاری است، پس اگر نبود خطا نگیر)
	fileHeader, err := c.FormFile("image")
	if err != nil {
		if !strings.HasPrefix(fileHeader.Header.Get("Content-Type"), "image/") {
			return fiber.NewError(fiber.StatusBadRequest, "Bad Request")
		}
	}
	// پارس کردن فرم (استفاده از FormValue های کاربر)
	// نکته: اگر فایل اجباری نیست، فقط اگر err == nil بود پردازش کن

	dto := service.CreateEventDTO{
		Title:       c.FormValue("title"),
		Description: c.FormValue("description"),
		Location:    c.FormValue("location"),
	}

	// ... بقیه منطق پارس کردن
	// اینجا منطق آپدیت را فراخوانی کن
	appErr := h.svc.UpdateEvent(c, id, fileHeader, dto)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "event updated"})
}

func (h *EventHandler) DeleteEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.svc.DeleteEvent(id)
	if err != nil {
		return c.Status(err.StatusCode).JSON(err)
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "event Deleted"})
}



// GetEvents (Public)
func (h *EventHandler) GetEvents(c *fiber.Ctx) error {
    page := c.QueryInt("page", 1)
    limit := c.QueryInt("limit", 10)
    search := c.Query("search", "")
    location := c.Query("location", "")

    // فراخوانی سرویس عمومی
    res, err := h.svc.GetPublicEvents(c.UserContext(), page, limit, search, location)
    if err != nil {
        return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch events"})
    }

    return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": res})
}

// ListEvents [Protected])
func (h *EventHandler) ListEvents(c *fiber.Ctx) error {
    page := c.QueryInt("page", 1)
    limit := c.QueryInt("limit", 10)
    search := c.Query("search", "")
    location := c.Query("location", "")

    // فراخوانی سرویس ادمین
    res, appErr := h.svc.GetAdminEvents(page, limit, search, location)
    if appErr != nil {
        return c.Status(appErr.StatusCode).JSON(appErr)
    }

    return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": res})
}

func (h *EventHandler) GetHomeData(c *fiber.Ctx) error {
    data, err := h.svc.GetHomeEvents()
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to load home data"})
    }
    return c.JSON(data)
}