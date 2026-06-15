package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
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
    Title       string                `form:"title"`       // فقط اطلاعاتی که کاربر مجاز است ویرایش کند
    Description string                `form:"description"`
    Location    string                `form:"location"`
    Image       *multipart.FileHeader `form:"image"`       // فایلی که می‌خواهیم آپلود کنیم
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
    // ۴. تبدیل زمان
    parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
    if err != nil {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date format"})
    }

	   file, err := c.FormFile("image") // 'image' اسم فیلد در فرم فرانت‌اند است
    var imageURL string
    if err == nil {
        // اعتبارسنجی نوع فایل (اختیاری ولی توصیه شده)
        if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
            return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Only images allowed"})
        }

        // تغییر نام برای جلوگیری از تداخل (استفاده از زمان و نام اصلی)
        filename := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
        savePath := filepath.Join("uploads", filename)
        
        if err := c.SaveFile(file, savePath); err != nil {
            return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Could not save image"})
        }
        imageURL = "/uploads/" + filename // مسیری که در دیتابیس ذخیره می‌کنی
    }

    // ۵. آماده‌سازی DTO (فیلد جدید ImageURL را اضافه کن)
    dto := service.CreateEventDTO{
        Title:       title,
        Description: description,
        Location:    location,
        StartTime:   parsedTime,
        Rows:        rows,
        SeatsPerRow: seatsPerRow,
        ImageUrl:    imageURL, // 👈 اینجا اضافه شد
    }

    event, appErr := h.svc.CreateNewEvent(dto)
    if appErr != nil {
        return c.Status(appErr.StatusCode).JSON(appErr)
    }

    return c.Status(http.StatusCreated).JSON(fiber.Map{
        "message":  "event created",
        "event_id": event.ID,
    })
}


func (h *EventHandler) GetEvents(c *fiber.Ctx) error {

	ctx := c.UserContext()

	events, source, err := h.svc.GetActiveEvents(ctx)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "خطا در دریافت لیست رویدادها"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"source": source, //
		"data":   events,
	})
}


