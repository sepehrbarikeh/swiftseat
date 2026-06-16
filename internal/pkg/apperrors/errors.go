package apperrors

import "fmt"


type AppError struct {
	StatusCode int    `json:"-"`      
	Message    string `json:"message"` 
	RawError   error  `json:"-"`       
}

func (e *AppError) Error() string {
	if e.RawError != nil {
		return fmt.Sprintf("%s | Raw Error: %v", e.Message, e.RawError)
	}
	return e.Message
}


func New(statusCode int, message string, rawErr error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		RawError:   rawErr,
	}
}


func NewValidationError(message string) *AppError {
	return &AppError{
		StatusCode: 400,
		Message:    message,
		RawError:   nil,
	}
}