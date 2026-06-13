package main

import (
	"fmt"
	"swift-seat/internal/config"
	"swift-seat/internal/repository"
)

func main() {
	fmt.Println("🔥 SwiftSeat is initializing...")

	// load settings from config.yaml
	cfg := config.LoadConfig("./")

	// Initialize the database connection
	db := repository.InitDB(cfg)

	fmt.Println("💾 Database connection established:", db != nil)

	fmt.Println("💎 The SwiftSeat engine is ready for use.")
	
}
