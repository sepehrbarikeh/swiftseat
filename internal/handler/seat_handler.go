package handlers

import (
	"net/http"
	"github.com/gofiber/fiber/v2"
	"swift-seat/internal/service"
)

type SeatHandler struct {
	svc *service.SeatService
}

func NewSeatHandler(svc *service.SeatService) *SeatHandler {
	return &SeatHandler{svc: svc}
}

// ساختار رکوئست را تمیز می‌کنیم (دیگر نیازی به user_id در بدنه نیست)
type ReserveSeatRequest struct {
	SeatID  uint `json:"seat_id"`
	EventID uint `json:"event_id"`
}

type ConfirmPaymentRequest struct {
	SeatID  uint  `json:"seat_id"`
	EventID uint  `json:"event_id"`
	Amount  int64 `json:"amount"`
}

func (h *SeatHandler) Reserve(c *fiber.Ctx) error {
	var req ReserveSeatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.SeatID == 0 || req.EventID == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "seat_id and event_id are required"})
	}

	// استخراج امنِ آی‌دی کاربر از میدل‌ور
	userID := c.Locals("userID").(uint)

	appErr := h.svc.HoldSeat(req.SeatID, req.EventID, userID)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Seat successfully reserved for 10 minutes",
	})
}

func (h *SeatHandler) ConfirmPayment(c *fiber.Ctx) error {
	var req ConfirmPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if req.SeatID == 0 || req.EventID == 0 || req.Amount <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "seat_id, event_id and amount are required"})
	}

	userID := c.Locals("userID").(uint)

	appErr := h.svc.ConfirmPayment(req.SeatID, req.EventID, userID, req.Amount)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Payment confirmed successfully and your ticket has been issued",
		"status":  "success",
	})
} 