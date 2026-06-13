package router

import (
	"swift-seat/internal/handler"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App,handler * handlers.EventHandler) {

	api := app.Group("/api")
	{
		api.Post("/events", handler.CreateEvent)
		api.Get("/events", handler.GetEvents)
	}
}
