// @title SwiftSeat API
// @version 1.0
// @description SwiftSeat REST API documentation
// @host localhost
// @BasePath /api
// @schemes http
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
package main

import (
	"fmt"
	"swift-seat/cmd/docs"
	"swift-seat/internal/config"
	"swift-seat/internal/database"
	handlers "swift-seat/internal/handler"
	"swift-seat/internal/middleware"
	token "swift-seat/internal/pkg/Token"
	"swift-seat/internal/pkg/worker"
	"swift-seat/internal/repository"
	"swift-seat/internal/server"
	"swift-seat/internal/service"
	"swift-seat/internal/sse"
	"sync"
)

func main() {
	fmt.Println("🔥 SwiftSeat is initializing...")

	var wg sync.WaitGroup
	// load settings from config.yaml
	cfg := config.LoadConfig("./")

	docs.SwaggerInfo.Title = "SwiftSeat API"
	docs.SwaggerInfo.Description = "SwiftSeat REST API documentation"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%s", cfg.AppPort)
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// Initialize the database connection
	db := repository.InitDB(cfg)

	db.SeedAdmin()

	token := token.New(cfg.JWTSecret)

	fmt.Println("💾 Database connection established:", db.DB)
	fmt.Println("💎 The SwiftSeat engine is ready for use.")
    
	rds := database.InitRedis(cfg)

	sseHub := sse.NewHub()
    
    // ۲. ساختِ هندلرها و تزریقِ Hub
	sseHandler := handlers.NewSSEHandler(sseHub)

	cronWorker := worker.NewCleanupWorker(db,cfg.CleanupInterval)
	cronWorker.Start()

	// user service
	userSvc := service.NewUserService(db, token)
	userHandler := handlers.NewUserHandler(userSvc)
	// event services
	eventSvc := service.NewEventService(db, &wg, rds)
	eventHandler := handlers.NewEventHandler(eventSvc)


	// seat service
	seatSvc := service.NewSeatService(db, cfg.SeetLock,sseHub)
	seetHandler := handlers.NewSeatHandler(seatSvc)


	authMiddlaware := middleware.NewAuthMiddleware(token)
	server := server.NewServer(cfg.AppPort, &wg, eventHandler, userHandler, seetHandler,sseHandler, authMiddlaware)
	server.StartServer()

}
