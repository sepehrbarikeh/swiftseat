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


// ConfirmPayment فرآیند پرداخت و خرید قطعی صندلی را مدیریت می‌کند
func (s *SeatService) ConfirmPayment(seatID, eventID, userID uint, amount int64) *apperrors.AppError {
	// شروع تراکنش دیتابیس
	err := s.repo.ConfirmPayment(seatID,eventID,userID,amount)
	// نگاشت خطاهای تراکنش به پاسخ‌های کاستوم API
	if err != nil {
		switch err.Error() {
		case "seat_not_found":
			return &apperrors.AppError{StatusCode: 404, Message: "صندلی مورد نظر یافت نشد"}
		case "not_your_reservation":
			return &apperrors.AppError{StatusCode: 400, Message: "این صندلی در رزرو شما نیست یا قبلاً فروخته شده است"}
		case "reservation_expired":
			return &apperrors.AppError{StatusCode: 410, Message: "مهلت ۱۰ دقیقه‌ای رزرو شما به پایان رسیده است"}
		default:
			return &apperrors.AppError{StatusCode: 500, Message: "خطای داخلی سرور در پردازش پرداخت"}
		}
	}

	return nil
}
