package apperrors

import "fmt"

// AppError ساختار خطای غنی برای کل سیستم
type AppError struct {
	StatusCode int    `json:"-"`       // کد HTTP مثل 400 یا 404 یا 500
	Message    string `json:"message"` // پیغام کاربرپسند
	RawError   error  `json:"-"`       // خطای واقعی دیتابیس یا سیستم برای لاگ داخلی
}

func (e *AppError) Error() string {
	if e.RawError != nil {
		return fmt.Sprintf("%s | Raw Error: %v", e.Message, e.RawError)
	}
	return e.Message
}

// New ساخت خطا با یک خطای خام دیگر
func New(statusCode int, message string, rawErr error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		RawError:   rawErr,
	}
}

// NewValidationError برای خطاهای اعتبارسنجی ورودی‌ها (Status 400)
func NewValidationError(message string) *AppError {
	return &AppError{
		StatusCode: 400,
		Message:    message,
		RawError:   nil,
	}
}