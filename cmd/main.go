package main

import (
	"fmt"
	"swift-seat/internal/config"
	handlers "swift-seat/internal/handler"
	"swift-seat/internal/middleware"
	token "swift-seat/internal/pkg/Token"
	"swift-seat/internal/pkg/worker"
	"swift-seat/internal/repository"
	"swift-seat/internal/server"
	"swift-seat/internal/service"
)

func main() {
	fmt.Println("🔥 SwiftSeat is initializing...")

	// load settings from config.yaml
	cfg := config.LoadConfig("./")

	// Initialize the database connection
	db := repository.InitDB(cfg)

	Token := token.New(cfg.JWTSecret)

	fmt.Println("💾 Database connection established:", db.DB)
	fmt.Println("💎 The SwiftSeat engine is ready for use.")

	cronWorker := worker.NewCleanupWorker(db, cfg.CleanupInterval)
	cronWorker.Start()

	eventSvc := service.NewEventService(db)
	eventHandler := handlers.NewEventHandler(eventSvc)

	seetSvc := service.NewSeatService(db, cfg.SeetLock)
	seetHandler := handlers.NewSeatHandler(seetSvc)

	authMiddlaware := middleware.NewAuthMiddleware(Token)
	server := server.NewServer(cfg.AppPort, eventHandler, seetHandler,authMiddlaware)
	server.StartServer()

}
