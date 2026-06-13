package router

import (
	"swift-seat/internal/handler"
	"swift-seat/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, eventHandler *handlers.EventHandler, seatHandler *handlers.SeatHandler,middleware *middleware.AuthMiddleware) {

	api := app.Group("/api")

	api.Get("/events", eventHandler.GetEvents)

	// روت‌های خصوصی (محافظت شده با میدل‌ورِ AuthRequired)
	secured := api.Group("/", middleware.AuthRequired())

	secured.Post("/events", eventHandler.CreateEvent)   // فقط کاربران لاگین شده ایونت بسازند
	secured.Post("/seats/reserve", seatHandler.Reserve) // رزرو صندلی کاملاً امن شد!
	secured.Post("/seats/confirm-payment", seatHandler.ConfirmPayment)


}
