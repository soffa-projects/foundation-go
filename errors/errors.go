package errors

import (
	"errors"
	"net/http"
)

type CustomError struct {
	Code    int
	Message string
}

// FIXED: Use pointer receiver to enable proper error comparison with errors.Is()
func (e *CustomError) Error() string {
	return e.Message
}

func Technical(message string) error {
	return &CustomError{
		Code:    http.StatusInternalServerError,
		Message: message,
	}
}

func BadRequest(message string) error {
	return &CustomError{
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

func Unauthorized(message string) error {
	return &CustomError{
		Code:    http.StatusUnauthorized,
		Message: message,
	}
}

func Forbidden(message string) error {
	return &CustomError{
		Code:    http.StatusForbidden,
		Message: message,
	}
}

func NotFound(message string) error {
	return &CustomError{
		Code:    http.StatusNotFound,
		Message: message,
	}
}

func Conflict(message string) error {
	return &CustomError{
		Code:    http.StatusConflict,
		Message: message,
	}
}

// GetStatusCode extracts HTTP status code from error
func GetStatusCode(err error) int {
	var ce *CustomError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return http.StatusInternalServerError
}

// Deprecated: Use errors.Is from standard library instead
func Is(err error, target error) bool {
	return errors.Is(err, target)
}
