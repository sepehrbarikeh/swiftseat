package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/pkg/utils"
	"swift-seat/internal/service"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CreateEventRequest struct {
    Title       string                `form:"title" validate:"required,min=3"`
    Description string                `form:"description" validate:"required"`
    Location    string                `form:"location" validate:"required"`
    StartTime   string                `form:"start_time" validate:"required"`
    Rows        int                   `form:"rows" validate:"required,gt=0"`
    SeatsPerRow int                   `form:"seats_per_row" validate:"required,gt=0"`
    Image       *multipart.FileHeader `form:"image" validate:"required"`
}

// @Summary Create an event
// @Description Create a new event in the system
// @Tags Events
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param title formData string true "Event title"
// @Param description formData string false "Event description"
// @Param location formData string true "Event location"
// @Param start_time formData string true "Event start time in RFC3339"
// @Param rows formData int true "Number of rows"
// @Param seats_per_row formData int true "Number of seats per row"
// @Param image formData file false "Event image"
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



	parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		appErr := apperrors.NewValidationError("Invalid date format")
		return c.Status(appErr.StatusCode).JSON(appErr)
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

	if errs := utils.ValidateStruct(dto); errs != nil {
        return c.Status(422).JSON(errs)
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

// @Summary Update an event
// @Description Update an existing event's metadata and optional image
// @Tags Events
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Event ID"
// @Param title formData string false "Event title"
// @Param description formData string false "Event description"
// @Param location formData string false "Event location"
// @Param image formData file false "Event image"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/events/{id} [put]
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

	if errs := utils.ValidateStruct(dto); errs != nil {
        return c.Status(422).JSON(errs)
    }

	// ... بقیه منطق پارس کردن
	// اینجا منطق آپدیت را فراخوانی کن
	appErr := h.svc.UpdateEvent(c, id, fileHeader, dto)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "event updated"})
}

// @Summary Delete an event
// @Description Delete an existing event by ID
// @Tags Events
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "Event ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/events/{id} [delete]
func (h *EventHandler) DeleteEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.svc.DeleteEvent(id)
	if err != nil {
		return c.Status(err.StatusCode).JSON(err)
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "event Deleted"})
}

// GetEvents (Public)
// @Summary List public events
// @Description List events with optional pagination and filtering
// @Tags Events
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(10)
// @Param search query string false "Search term"
// @Param location query string false "Location filter"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/events [get]
func (h *EventHandler) GetEvents(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	search := c.Query("search", "")
	location := c.Query("location", "")

	// فراخوانی سرویس عمومی
	res, appErr := h.svc.GetPublicEvents(c.UserContext(), page, limit, search, location)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": res})
}

// ListEvents [Protected])
// @Summary List admin events
// @Description List events with admin-only details and pagination
// @Tags Events
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(10)
// @Param search query string false "Search term"
// @Param location query string false "Location filter"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/events/all [get]
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

// @Summary Get API home data
// @Description Retrieve the home dashboard payload for public display
// @Tags Public
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router / [get]
func (h *EventHandler) GetHomeData(c *fiber.Ctx) error {
	data, appErr := h.svc.GetHomeEvents()
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}
	return c.JSON(data)
}
