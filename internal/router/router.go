package router

import (
	"fmt"
	"swift-seat/internal/handler"
	"swift-seat/internal/middleware"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func SetupRoutes(app *fiber.App, eventHandler *handlers.EventHandler, seatHandler *handlers.SeatHandler, userHandler handlers.UserHandler, middleware *middleware.AuthMiddleware) {

	seatLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Restrict based on the logged-in user ID (which we got from the JWT middleware)
			// This way, even if a user comes with multiple devices, they will still be limited.
			if userID := c.Locals("userID"); userID != nil {
				return fmt.Sprintf("user_%v", userID)
			}
			return c.IP() // Restrict by ip
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Request limited",
			})
		},
	})

	// Swagger UI
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	api := app.Group("/api")
	api.Get("/",eventHandler.GetHomeData)

	api.Get("/events/:event_id/seats", seatHandler.GetSeatMap)

	// user routes //
	api.Post("/register", userHandler.Register)
	api.Post("/login", userHandler.Login)

	// events routes //
	api.Get("/events",eventHandler.GetEvents)

	secured := api.Group("/", middleware.AuthRequired)

	// seats routes
	secured.Post("/seats/reserve", seatLimiter, seatHandler.Reserve)
	secured.Post("/seats/confirm-payment", seatHandler.ConfirmPayment)
	secured.Get("/user/tickets", seatHandler.GetMyTickets)

	// admin routes
	admin := secured.Group("/", middleware.AdminOnly)
	admin.Post("/users/:id/role", userHandler.ChangeUserRole)
	admin.Post("/events", eventHandler.CreateEvent)

	admin.Put("/events/:id", eventHandler.UpdateEvent)
	admin.Delete("/events/:id", eventHandler.DeleteEvent)
	admin.Get("/tickets/validate/:ref", seatHandler.ValidateTicket)
	admin.Get("/events/all",eventHandler.ListEvents)
}
