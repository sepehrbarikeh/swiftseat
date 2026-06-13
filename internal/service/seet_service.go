package service

import (
	"net/http"
	"time"

	"swift-seat/internal/pkg/apperrors"
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