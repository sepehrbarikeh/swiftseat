package service

import (
	"encoding/json"
	"time"

	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
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

func NewSeatService(repo *repository.PostgresDB, seatLockDuration time.Duration, hub *sse.Hub) *SeatService {
	return &SeatService{repo: repo,
		seatLockDuration: seatLockDuration,
		hub:              hub,
	}
}

func (s *SeatService) HoldSeat(SeatNumber []string, eventID uint, userID uint) (string,*apperrors.AppError) {

	lockDuration := s.seatLockDuration

	reserv, appErr := s.repo.CreateReservation(SeatNumber, eventID, userID, lockDuration);
	if appErr != nil {
		return "",appErr
	}

	

	msgBytes, err := json.Marshal(reserv)
	if err == nil {
		s.hub.Broadcast(msgBytes)
	}

	return reserv.Ref, nil
}

func (s *SeatService) ConfirmPayment(TicketRef string, eventID, userID uint, amount int64) (*models.Ticket, *apperrors.AppError) {


	ticket, appErr := s.repo.ExecutePaymentTransaction(TicketRef, userID, amount)
	if appErr != nil {
		return nil, appErr
	}

	msgData := map[string]interface{}{
		"event_id":    eventID,
		"seat_number": ticket.SeatID, 
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
