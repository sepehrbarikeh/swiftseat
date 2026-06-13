package models

import (
	"time"

	"gorm.io/gorm"
)

// User مدل کاربر
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(100);not null" json:"name"`
	Email        string    `gorm:"type:varchar(150);unique;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Role         string    `gorm:"type:varchar(20);default:'customer'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Event مدل رویداد/کنسرت
type Event struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Location    string    `gorm:"type:varchar(200);not null" json:"location"`
	StartTime   time.Time `gorm:"not null" json:"start_time"`
	TotalSeats  int       `gorm:"not null" json:"total_seats"`
	CreatedAt   time.Time `json:"created_at"`
	Seats       []Seat    `json:"seats,omitempty"`
	Status    string    `gorm:"type:varchar(30);default:'creating_seats';not null" json:"status"`
}

// Seat مدل صندلی‌های فیزیکی سالن
type Seat struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	EventID    uint    `gorm:"not null" json:"event_id"`
	SeatNumber string  `gorm:"type:varchar(10);not null" json:"seat_number"`
	RowName    string  `gorm:"type:varchar(10);not null" json:"row_name"`
	Price      float64 `gorm:"type:numeric(10,2);not null" json:"price"`
}

type SeatStatus struct {
	ID uint `gorm:"primaryKey" json:"id"`

	SeatID  uint `gorm:"uniqueIndex:idx_seat_event;not null" json:"seat_id"`
	EventID uint `gorm:"uniqueIndex:idx_seat_event;not null" json:"event_id"`

	Status     string     `gorm:"type:varchar(20);default:'available'" json:"status"` // available, reserved, sold
	ReservedBy *uint      `json:"reserved_by,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// Booking مدل بلیط صادر شده قطعی
type Booking struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"not null" json:"user_id"`
	EventID       uint      `gorm:"not null" json:"event_id"`
	SeatID        uint      `gorm:"not null" json:"seat_id"`
	PaymentStatus string    `gorm:"type:varchar(20);default:'paid'" json:"payment_status"`
	CreatedAt     time.Time `json:"created_at"`
}

type Ticket struct {
    ID uint `gorm:"primaryKey" json:"id"`

    // 🚀 فیلد SeatID باید برگردد تا رابطه دیتابیسی برقرار شود
    SeatID     uint `gorm:"not null" json:"seat_id"`
    Seat       Seat `gorm:"foreignKey:SeatID" json:"seat"` 

    EventID    uint  `gorm:"not null" json:"event_id"`
    Event      Event `gorm:"foreignKey:EventID" json:"event"` 

    UserID     uint           `gorm:"not null" json:"user_id"`
    TicketRef  string         `gorm:"type:varchar(100);unique;not null" json:"ticket_ref"`
    PaidAmount int64          `json:"paid_amount"`
    CreatedAt  time.Time      `json:"created_at"`
    DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}