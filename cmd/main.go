package main

import (
	"fmt"
	"swift-seat/internal/config"
	"swift-seat/internal/database"
	handlers "swift-seat/internal/handler"
	"swift-seat/internal/middleware"
	token "swift-seat/internal/pkg/Token"
	"swift-seat/internal/pkg/worker"
	"swift-seat/internal/repository"
	"swift-seat/internal/server"
	"swift-seat/internal/service"
	"sync"
)

func main() {
	fmt.Println("🔥 SwiftSeat is initializing...")

	var wg sync.WaitGroup
	// load settings from config.yaml
	cfg := config.LoadConfig("./")

	// Initialize the database connection
	db := repository.InitDB(cfg)

	Token := token.New(cfg.JWTSecret)

	fmt.Println(Token.GenerateToken(42))

	fmt.Println("💾 Database connection established:", db.DB)
	fmt.Println("💎 The SwiftSeat engine is ready for use.")

	rds := database.InitRedis()


	cronWorker := worker.NewCleanupWorker(db, cfg.CleanupInterval)
	cronWorker.Start()

	eventSvc := service.NewEventService(db, &wg,rds)
	eventHandler := handlers.NewEventHandler(eventSvc)

	seatSvc := service.NewSeatService(db, cfg.SeetLock)
	seetHandler := handlers.NewSeatHandler(seatSvc)

	authMiddlaware := middleware.NewAuthMiddleware(Token)
	server := server.NewServer(cfg.AppPort, &wg, eventHandler, seetHandler, authMiddlaware)
	server.StartServer()

}
