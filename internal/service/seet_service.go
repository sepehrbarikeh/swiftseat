package service

import (
	"encoding/json"
	"time"

	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/pkg/ticket"
	"swift-seat/internal/repository"
	"swift-seat/internal/sse"
)

type SeatService struct {
	repo             *repository.PostgresDB
	hub              *sse.Hub
	seatLockDuration time.Duration
}

type SeatResponseDTO struct {
	SeatID     uint    `json:"seat_id"`
	SeatNumber string  `json:"seat_number"`
	RowName    string  `json:"row_name"`
	Price      float64 `json:"price"`
	Status     string  `json:"status"`
}

type PaginatedEventsResponse struct {
	TotalItems  int64          `json:"total_items"`
	TotalPages  int            `json:"total_pages"`
	CurrentPage int            `json:"current_page"`
	Limit       int            `json:"limit"`
	Events      []models.Event `json:"events"`
}

func NewSeatService(repo *repository.PostgresDB, seatLockDuration time.Duration, hub *sse.Hub) *SeatService {
	return &SeatService{repo: repo,
		seatLockDuration: seatLockDuration,
		hub:              hub,
	}
}

func (s *SeatService) HoldSeat(SeatNumber string, eventID uint, userID uint) *apperrors.AppError {

	lockDuration := s.seatLockDuration

	if appErr := s.repo.ReserveSeatWithLock(SeatNumber, eventID, userID, lockDuration); appErr != nil {
		return appErr
	}

	msgData := map[string]interface{}{
		"event_id":    eventID,
		"seat_number": SeatNumber,
		"new_status":  "reserved",
	}

	msgBytes, err := json.Marshal(msgData)
	if err == nil {
		s.hub.Broadcast(msgBytes)
	}

	return nil
}

func (s *SeatService) ConfirmPayment(SeatNumber string, eventID, userID uint, amount int64) (*models.Ticket, *apperrors.AppError) {

	ticketRef := ticket.GenerateTicketRef()

	ticket, appErr := s.repo.ExecutePaymentTransaction(SeatNumber, eventID, userID, amount, ticketRef)
	if appErr != nil {
		return nil, appErr
	}

	msgData := map[string]interface{}{
		"event_id":    eventID,
		"seat_number": SeatNumber, 
		"new_status":  "sold",
	}

	// ۳. تبدیل به JSON
	msgBytes, err := json.Marshal(msgData)
	if err == nil {
		s.hub.Broadcast(msgBytes)
	}

	return ticket, nil
}

func (s *SeatService) GetUserTickets(userID uint) ([]models.Ticket, *apperrors.AppError) {
	tickets, appErr := s.repo.GetUserTickets(userID)
	if appErr != nil {
		return nil, appErr
	}
	return tickets, nil
}

func (s *SeatService) GetEventSeatMap(eventID uint) ([]SeatResponseDTO, *apperrors.AppError) {
	statuses, appErr := s.repo.GetEventSeatsWithStatus(eventID)
	if appErr != nil {
		return nil, appErr
	}

	var seatMap []SeatResponseDTO
	now := time.Now()

	for _, st := range statuses {
		
		finalStatus := st.Status

	
		if st.Status == "reserved" {
			if st.ExpiresAt != nil && st.ExpiresAt.Before(now) {
				finalStatus = "available"
			} else {
				
				finalStatus = "reserved"
			}
		}

		
		seatMap = append(seatMap, SeatResponseDTO{
			SeatID:     st.SeatID,
			SeatNumber: st.Seat.SeatNumber,
			RowName:    st.Seat.RowName,
			Price:      st.Seat.Price,
			Status:     finalStatus, 
		})
	}

	return seatMap, nil
}

func (s *SeatService) GetTicketByRef(ref string) (*models.Ticket, *apperrors.AppError) {
	ticket, appErr := s.repo.GetTicketByRef(ref)
	if appErr != nil {
		return nil, appErr
	}
	return &ticket, nil
}
