package server

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"swift-seat/internal/handler"
	"swift-seat/internal/middleware"
	"swift-seat/internal/router"
	"sync"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type Server struct {
	port       string
	wg   *sync.WaitGroup 
	eventHandler *handlers.EventHandler
	seatHandler *handlers.SeatHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewServer(port string,wg   *sync.WaitGroup ,eventHandler *handlers.EventHandler,seatHandler *handlers.SeatHandler,authMiddleware *middleware.AuthMiddleware) *Server {
	return &Server{
		port: port,
		wg: wg,
		eventHandler: eventHandler,
		seatHandler: seatHandler,
		authMiddleware: authMiddleware,
	}
}


func (s *Server) StartServer() {

	shutdownChan := make(chan os.Signal, 1)
    signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	router.SetupRoutes(app,s.eventHandler,s.seatHandler,s.authMiddleware)
	
	fmt.Println("💎 SwiftSeat Server is running on port",s.port)
	go func() {
        if err := app.Listen(":" + s.port); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()

	<-shutdownChan
    log.Println("Shutting down Fiber server...")

	
    if err := app.Shutdown(); err != nil {
        log.Printf("Fiber shutdown error: %v", err)
    }

    
    log.Println("Waiting for background tasks to complete safely...")
    s.wg.Wait() 

    log.Println("All systems cleanly stopped. Goodbye!")
}