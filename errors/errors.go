package errors

import (
	"errors"
	"net/http"
)

type CustomError struct {
	Code    int
	Message string
}

func (e CustomError) Error() string {
	return e.Message
}

func Technical(message string) error {
	return CustomError{
		Code:    http.StatusInternalServerError,
		Message: message,
	}
}

func BadRequest(message string) error {
	return CustomError{
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

func Unauthorized(message string) error {
	return CustomError{
		Code:    http.StatusUnauthorized,
		Message: message,
	}
}

func Forbidden(message string) error {
	return CustomError{
		Code:    http.StatusForbidden,
		Message: message,
	}
}

func NotFound(message string) error {
	return CustomError{
		Code:    http.StatusNotFound,
		Message: message,
	}
}

func Conflict(message string) error {
	return CustomError{
		Code:    http.StatusConflict,
		Message: message,
	}
}

func Is(err error, target error) bool {
	return errors.Is(err, target)
}
