package server

import (
	"fmt"
	"log"
	"swift-seat/internal/handler"
	"swift-seat/internal/middleware"
	"swift-seat/internal/router"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type Server struct {
	port       string
	eventHandler *handlers.EventHandler
	seatHandler *handlers.SeatHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewServer(port string,eventHandler *handlers.EventHandler,seatHandler *handlers.SeatHandler,authMiddleware *middleware.AuthMiddleware) *Server {
	return &Server{
		port: port,
		eventHandler: eventHandler,
		seatHandler: seatHandler,
		authMiddleware: authMiddleware,
	}
}


func (s *Server) StartServer() {

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	router.SetupRoutes(app,s.eventHandler,s.seatHandler,s.authMiddleware)
	
	
	fmt.Println("💎 SwiftSeat Server is running on port",s.port)
	log.Fatal(app.Listen(":" + s.port))
}