package main

import (
	"fmt"
	"swift-seat/internal/config"
	handlers "swift-seat/internal/handler"
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

	fmt.Println("💾 Database connection established:", db.DB)

	fmt.Println("💎 The SwiftSeat engine is ready for use.")

	eventSvc := service.NewEventService(db)
	eventHandler := handlers.NewEventHandler(eventSvc)
	server := server.NewServer(cfg.AppPort,eventHandler)
	server.StartServer()

	
}
