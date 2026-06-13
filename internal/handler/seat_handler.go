package handlers

import (
	"fmt"
	"net/http"
	"swift-seat/internal/service"

	"github.com/gofiber/fiber/v2"
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

    // 🚀 این خط جا افتاده بود؛ برای خواندن دیتای ارسالی از کلاینت
    if err := c.BodyParser(&req); err != nil {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Bad request"})
    }

    if req.SeatID == 0 || req.EventID == 0 {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "seat_id and event_id are required"})
    }

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
		fmt.Println(req)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}


	userID := c.Locals("userID").(uint)

	// صدا زدن سرویس اصلاح شده
	ticket, appErr := h.svc.ConfirmPayment(req.SeatID, req.EventID, userID, req.Amount)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	// برگرداندن شناسنامه کامل بلیت به کلاینت
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Payment has been successfully.",
		"ticket": fiber.Map{
			"ticket_ref":   ticket.TicketRef,   
			"paid_amount":  ticket.PaidAmount,  
			"issued_at":    ticket.CreatedAt,   
			"seat_number":  ticket.Seat.ID, 
			"row_number":   ticket.Seat.RowName,   
			"event_title":  ticket.Event.Title, 
			"singer_name":  ticket.Event.Title,
			"hall_name":    ticket.Event.Location, 
		},
	})
}

