package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"swift-seat/internal/service"
)



type CreateEventRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Location     string `json:"location"`
	StartTime    string `json:"start_time"`
	Rows         int    `json:"rows"`
	SeatsPerRow  int    `json:"seats_per_row"`
}

func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	var req CreateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	dto := service.CreateEventDTO{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   req.StartTime,
		Rows:        req.Rows,
		SeatsPerRow: req.SeatsPerRow,
	}

	event, appErr := h.svc.CreateNewEvent(dto)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "event created",
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