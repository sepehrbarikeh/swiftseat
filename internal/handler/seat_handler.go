package handlers

import (
	"fmt"
	"net/http"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/pkg/utils"
	"swift-seat/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SeatHandler struct {
	svc *service.SeatService
}

func NewSeatHandler(svc *service.SeatService) *SeatHandler {
	return &SeatHandler{svc: svc}
}

type ReserveSeatRequest struct {
	SeatNumber []string `json:"seat_number" validate:"required"`
	EventID    uint     `json:"event_id" validate:"required,gt=0"`
}

type ConfirmPaymentRequest struct {
	TicketRef string `json:"ticket_ref" validate:"required"`
	EventID   uint   `json:"event_id" validate:"required,gt=0"`
	Amount    int64  `json:"amount" validate:"required,gt=0"`
}

// @Summary Reserve a seat
// @Description Temporarily reserve a seat before payment
// @Tags Seats
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param reservation body ReserveSeatRequest true "Seat reservation"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/seats/reserve [post]
func (h *SeatHandler) Reserve(c *fiber.Ctx) error {
	var req ReserveSeatRequest

	if err := c.BodyParser(&req); err != nil {
		appErr := apperrors.NewValidationError("Invalid request body")
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	if errs := utils.ValidateStruct(req); errs != nil {
		return c.Status(422).JSON(errs)
	}

	userID := c.Locals("user_id").(uint)

	ref, appErr := h.svc.HoldSeat(req.SeatNumber, req.EventID, userID)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message":    "Seat successfully reserved for 10 minutes",
		"ticket_ref": ref,
	})
}

// @Summary Confirm payment
// @Description Confirm payment for a reserved seat and issue a ticket
// @Tags Seats
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param payment body ConfirmPaymentRequest true "Payment confirmation"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/seats/confirm-payment [post]
func (h *SeatHandler) ConfirmPayment(c *fiber.Ctx) error {
	var req ConfirmPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Println(err)
		appErr := apperrors.NewValidationError("Invalid request body")
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	if errs := utils.ValidateStruct(req); errs != nil {
		return c.Status(422).JSON(errs)
	}

	userID := c.Locals("user_id").(uint)

	ticket, appErr := h.svc.ConfirmPayment(req.TicketRef, req.EventID, userID, req.Amount)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Payment has been successful.",
		"ticket": fiber.Map{
			"ticket_ref":  ticket.TicketRef,
			"paid_amount": ticket.PaidAmount,
			"issued_at":   ticket.CreatedAt,
			"seat_number": ticket.Seat.SeatNumber,
			"row_number":  ticket.Seat.RowName,
			"event_title": ticket.Event.Title,
			"singer_name": ticket.Event.Title,
			"hall_name":   ticket.Event.Location,
		},
	})
}

// @Summary Get current user's tickets
// @Description Retrieve tickets for the authenticated user
// @Tags Tickets
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/user/tickets [get]
func (h *SeatHandler) GetMyTickets(c *fiber.Ctx) error {

	userID := c.Locals("userID").(uint)

	tickets, appErr := h.svc.GetUserTickets(userID)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	var formattedTickets []fiber.Map
	for _, t := range tickets {
		formattedTickets = append(formattedTickets, fiber.Map{
			"ticket_ref":  t.TicketRef,
			"paid_amount": t.PaidAmount,
			"issued_at":   t.CreatedAt,
			"seat_number": t.Seat.SeatNumber,
			"row_name":    t.Seat.RowName,
			"event_title": t.Event.Title,
			"location":    t.Event.Location,
			"event_id":    t.Event.ID,
			"event_date":  t.Event.CreatedAt,
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"count":   len(formattedTickets),
		"tickets": formattedTickets,
	})
}

// @Summary Get seat map
// @Description Get the seat map for an event
// @Tags Seats
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/events/{event_id}/seats [get]
func (h *SeatHandler) GetSeatMap(c *fiber.Ctx) error {
	eventID, err := c.ParamsInt("event_id")
	if err != nil || eventID <= 0 {
		appErr := apperrors.NewValidationError("Invalid event id")
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	seatMap, appErr := h.svc.GetEventSeatMap(uint(eventID))
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":   "success",
		"event_id": eventID,
		"seats":    seatMap,
	})
}

// @Summary Validate a ticket
// @Description Validate a ticket reference and return ticket details
// @Tags Tickets
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param ref path string true "Ticket reference"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/tickets/validate/{ref} [get]
// ادمین کد تیکت را می‌فرستد و ما وضعیتش را چک می‌کنیم
func (h *SeatHandler) ValidateTicket(c *fiber.Ctx) error {
	ref := c.Params("ref")

	ticket, appErr := h.svc.GetTicketByRef(ref)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.JSON(fiber.Map{
		"status": "valid",
		"ticket": fiber.Map{
			"id":          ticket.ID,
			"event":       ticket.Event.Title,
			"seat":        ticket.Seat.SeatNumber,
			"owner_id":    ticket.UserID,
			"paid_amount": ticket.PaidAmount,
		},
	})
}
