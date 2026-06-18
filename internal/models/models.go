package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(100);not null" json:"name"`
	Email        string    `gorm:"type:varchar(150);unique;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Role         string    `gorm:"type:varchar(20);default:'user'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}


type Event struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	ImageURL    string `gorm:"size:255"`
	Location    string    `gorm:"type:varchar(200);not null" json:"location"`
	StartTime   time.Time `gorm:"not null" json:"start_time"`
	TotalSeats  int       `gorm:"not null" json:"total_seats"`
	CreatedAt   time.Time `json:"created_at"`
	Seats       []Seat    `json:"seats,omitempty"`
	Status    string    `gorm:"type:varchar(30);default:'creating_seats';not null" json:"status"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Seat struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	EventID    uint    `gorm:"not null" json:"event_id"`
	SeatNumber string  `gorm:"type:varchar(10);not null" json:"seat_number"`
	RowName    string  `gorm:"type:varchar(10);not null" json:"row_name"`
	Price      float64 `gorm:"type:numeric(10,2);not null" json:"price"`
}

type SeatStatus struct {
    gorm.Model
    EventID   uint       `gorm:"not null"`
    SeatID    uint       `gorm:"not null"`
    Seat      Seat       `gorm:"foreignKey:SeatID"` 
    Status    string     `gorm:"type:varchar(20);default:'available'"` // available, reserved, sold
    ReservedBy *uint
    ExpiresAt  *time.Time
}

type Ticket struct {
    ID uint `gorm:"primaryKey" json:"id"`

   
    SeatID     uint `gorm:"not null" json:"seat_id"`
    Seat       Seat `gorm:"foreignKey:SeatID" json:"seat"` 

    EventID    uint  `gorm:"not null" json:"event_id"`
    Event      Event `gorm:"foreignKey:EventID" json:"event"` 

    UserID     uint           `gorm:"not null" json:"user_id"`
    TicketRef  string         `gorm:"type:varchar(100);not null" json:"ticket_ref"`
    PaidAmount int64          `json:"paid_amount"`
    CreatedAt  time.Time      `json:"created_at"`
    DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type Reservation struct {
	ID        uint              `gorm:"primaryKey"`

	Ref       string            `gorm:"not null"`

	UserID    uint              `gorm:"index;not null"`
	EventID   uint              `gorm:"index:idx_user_event;not null"`

	Status    ReservationStatus `gorm:"type:varchar(20);index" json:"status"`

	TotalAmount int64           `gorm:"not null"`

	ExpiresAt *time.Time        `gorm:"index"`

	Version   int               `gorm:"default:1"`

	CreatedAt time.Time
	UpdatedAt time.Time

	DeletedAt gorm.DeletedAt    `gorm:"index"`
}

type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "pending"
	ReservationReserved  ReservationStatus = "reserved"
	ReservationPaid      ReservationStatus = "paid"
	ReservationExpired   ReservationStatus = "expired"
	ReservationCancelled ReservationStatus = "cancelled"
)

type ReservationSeat struct {
	ID            uint `gorm:"primaryKey"`

	ReservationID uint `gorm:"index;not null"`
	SeatID        uint `gorm:"index;not null"`

	CreatedAt time.Time
}