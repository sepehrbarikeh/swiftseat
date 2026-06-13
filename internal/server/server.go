package server

import (
	"fmt"
	"log"
	"swift-seat/internal/router"
	"swift-seat/internal/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type Server struct {
	port       string
	handler *handlers.EventHandler
}

func NewServer(port string,handler *handlers.EventHandler) *Server {
	return &Server{
		port: port,
		handler: handler,
	}
}


func (s *Server) StartServer() {

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	router.SetupRoutes(app,s.handler)
	
	
	fmt.Println("💎 SwiftSeat Server is running on port",s.port)
	log.Fatal(app.Listen(":" + s.port))
}