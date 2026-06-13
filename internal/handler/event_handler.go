package handlers

import (
	"net/http"
	"time"

	"swift-seat/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CreateEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Location    string `json:"location"`
	StartTime   string `json:"start_time"`
	Rows        int    `json:"rows"`
	SeatsPerRow int    `json:"seats_per_row"`
}

func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	var req CreateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}



	if req.Title == "" || req.Location == "" {
	return c.Status(http.StatusBadRequest).JSON(fiber.Map{
		"error":  "Event title and location cannot be empty",
	})
	}
	if req.Rows <= 0 || req.SeatsPerRow <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
		"error":  "Number of rows and seats must be greater than zero",
	})
	}

	parsedTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		 return c.Status(http.StatusBadRequest).JSON(fiber.Map{
		"error":  "Invalid date format",
	})
	}

	if parsedTime.Before(time.Now()) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
		"error":  "he time of the event cannot be in the past.",
	})
	}
	dto := service.CreateEventDTO{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   parsedTime,
		Rows:        req.Rows,
		SeatsPerRow: req.SeatsPerRow,
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
	events, appErr := h.svc.ListAllEvents()
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}
	return c.JSON(events)
}
