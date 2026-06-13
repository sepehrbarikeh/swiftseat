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



func NewSeatService(repo *repository.PostgresDB, seatLockDuration time.Duration) *SeatService {
	return &SeatService{repo: repo, seatLockDuration: seatLockDuration}
}

func (s *SeatService) HoldSeat(seatID uint, eventID uint, userID uint) *apperrors.AppError {

	lockDuration := s.seatLockDuration

	err := s.repo.ReserveSeatWithLock(seatID, eventID, userID, lockDuration)
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


func (s *SeatService) ConfirmPayment(seatID, eventID, userID uint, amount int64) (*models.Ticket, *apperrors.AppError) {

	ticketRef := ticket.GenerateTicketRef()

	
	ticket, err := s.repo.ExecutePaymentTransaction(seatID, eventID, userID, amount, ticketRef)
	
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

