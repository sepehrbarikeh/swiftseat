package repository

import (
	"fmt"
	"log"

	"swift-seat/internal/config"
	"swift-seat/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDB struct{
	DB *gorm.DB
}



// InitDB connects to the PostgreSQL database and performs auto-migration for the models.
func InitDB(config *config.Config) *PostgresDB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		config.DBHost,
		config.DBUser,
		config.DBPassword,
		config.DBName,
		config.DBPort,
		config.DBSSLMode,
		"Asia/Tehran")

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Database connection error %v", err)
	}

	fmt.Println("✅ The connection to PostgreSQL was successfully.")

	
	err = database.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.Seat{},
		&models.SeatStatus{},
		&models.Ticket{},
		&models.Reservation{},
		&models.ReservationSeat{},
	)
	if err != nil {
		log.Fatalf("❌ Migration error: %v", err)
	}

	fmt.Println("🚀 Tables migrated successfully.")
	return &PostgresDB{
		DB: database,
	}

}