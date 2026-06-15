package service

import (
	"net/http"
	"time"

	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/pkg/ticket"
	"swift-seat/internal/repository"
)

type SeatService struct {
	repo *repository.PostgresDB
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

func NewSeatService(repo *repository.PostgresDB, seatLockDuration time.Duration) *SeatService {
	return &SeatService{repo: repo, seatLockDuration: seatLockDuration}
}

func (s *SeatService) HoldSeat(SeatNumber string, eventID uint, userID uint) *apperrors.AppError {

	lockDuration := s.seatLockDuration

	err := s.repo.ReserveSeatWithLock(SeatNumber, eventID, userID, lockDuration)
	if err != nil {

		switch err.Error() {
		case "seat_already_sold":
			return apperrors.New(http.StatusConflict, "This seat has already been sold", err)
		case "seat_already_reserved":
			return apperrors.New(http.StatusConflict, "This seat is already reserved by another user", err)
		case "gorm: record not found":
			return apperrors.New(http.StatusNotFound, "Seat or event not found", err)
		default:
			return apperrors.New(http.StatusInternalServerError, "Unexpected error occurred on the server while reserving the seat", err)
		}
	}

	return nil
}


func (s *SeatService) ConfirmPayment(SeatNumber string, eventID, userID uint, amount int64) (*models.Ticket, *apperrors.AppError) {

	ticketRef := ticket.GenerateTicketRef()

	
	ticket, err := s.repo.ExecutePaymentTransaction(SeatNumber, eventID, userID, amount, ticketRef)
	
	if err != nil {
		switch err.Error() {
		case "seat_not_found":
			return nil, &apperrors.AppError{StatusCode: 404, Message: "Not found"}
		case "not_your_reservation":
			return nil, &apperrors.AppError{StatusCode: 400, Message: "This seat is not reserved by you or has already been sold"}
		case "reservation_expired":
			return nil, &apperrors.AppError{StatusCode: 410, Message: "The reservation time limit of 10 minutes has expired. Please reserve the seat again."}
		default:
			return nil, &apperrors.AppError{StatusCode: 500, Message: "Internal error"}
		}
	}

	return ticket, nil
}


func (s *SeatService) GetUserTickets(userID uint) ([]models.Ticket, *apperrors.AppError) {
    tickets, err := s.repo.GetUserTickets(userID)
    if err != nil {
        return nil, apperrors.New(http.StatusInternalServerError, "Failed to retrieve user tickets", err)
    }
    return tickets, nil
}


func (s *SeatService) GetEventSeatMap(eventID uint) ([]SeatResponseDTO, *apperrors.AppError) {
    statuses, err := s.repo.GetEventSeatsWithStatus(eventID)
    if err != nil {
        return nil, apperrors.New(http.StatusInternalServerError, "Failed to fetch seat map", err)
    }

    var seatMap []SeatResponseDTO
    now := time.Now()

    for _, st := range statuses {
        currentStatus := st.Status

        if st.Status == "reserved" && st.ExpiresAt != nil && st.ExpiresAt.Before(now) {
            currentStatus = "available"
        }

       

        seatMap = append(seatMap, SeatResponseDTO{
            SeatID:     st.SeatID,
            SeatNumber: st.Seat.SeatNumber,
            RowName:    st.Seat.RowName,
            Price:      st.Seat.Price,
            Status:     currentStatus,
        })
    }

    return seatMap, nil
}


func (s *SeatService) GetTicketByRef(ref string) (*models.Ticket, error) {
	ticket, err := s.repo.GetTicketByRef(ref)
	return &ticket, err
}