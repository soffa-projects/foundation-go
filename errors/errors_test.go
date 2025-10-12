package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestCustomError_Error(t *testing.T) {
	err := &CustomError{
		Code:    http.StatusBadRequest,
		Message: "test error message",
	}
	assert.Equal(t, err.Error(), "test error message")
}

func TestTechnical(t *testing.T) {
	err := Technical("internal error")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "internal error")

	// Verify it's a CustomError with correct code
	var ce *CustomError
	assert.Equal(t, errors.As(err, &ce), true)
	assert.Equal(t, ce.Code, http.StatusInternalServerError)
	assert.Equal(t, ce.Message, "internal error")
}

func TestBadRequest(t *testing.T) {
	err := BadRequest("invalid input")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "invalid input")

	var ce *CustomError
	assert.Equal(t, errors.As(err, &ce), true)
	assert.Equal(t, ce.Code, http.StatusBadRequest)
	assert.Equal(t, ce.Message, "invalid input")
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("authentication required")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "authentication required")

	var ce *CustomError
	assert.Equal(t, errors.As(err, &ce), true)
	assert.Equal(t, ce.Code, http.StatusUnauthorized)
	assert.Equal(t, ce.Message, "authentication required")
}

func TestForbidden(t *testing.T) {
	err := Forbidden("access denied")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "access denied")

	var ce *CustomError
	assert.Equal(t, errors.As(err, &ce), true)
	assert.Equal(t, ce.Code, http.StatusForbidden)
	assert.Equal(t, ce.Message, "access denied")
}

func TestNotFound(t *testing.T) {
	err := NotFound("resource not found")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "resource not found")

	var ce *CustomError
	assert.Equal(t, errors.As(err, &ce), true)
	assert.Equal(t, ce.Code, http.StatusNotFound)
	assert.Equal(t, ce.Message, "resource not found")
}

func TestConflict(t *testing.T) {
	err := Conflict("resource already exists")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "resource already exists")

	var ce *CustomError
	assert.Equal(t, errors.As(err, &ce), true)
	assert.Equal(t, ce.Code, http.StatusConflict)
	assert.Equal(t, ce.Message, "resource already exists")
}

func TestGetStatusCode_WithCustomError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"Technical", Technical("error"), http.StatusInternalServerError},
		{"BadRequest", BadRequest("error"), http.StatusBadRequest},
		{"Unauthorized", Unauthorized("error"), http.StatusUnauthorized},
		{"Forbidden", Forbidden("error"), http.StatusForbidden},
		{"NotFound", NotFound("error"), http.StatusNotFound},
		{"Conflict", Conflict("error"), http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetStatusCode(tt.err)
			assert.Equal(t, code, tt.expected)
		})
	}
}

func TestGetStatusCode_WithStandardError(t *testing.T) {
	err := errors.New("standard error")
	code := GetStatusCode(err)
	assert.Equal(t, code, http.StatusInternalServerError)
}

func TestGetStatusCode_WithNil(t *testing.T) {
	code := GetStatusCode(nil)
	assert.Equal(t, code, http.StatusInternalServerError)
}

func TestCustomError_ErrorsAs(t *testing.T) {
	err := NotFound("resource not found")

	var ce *CustomError
	assert.Equal(t, errors.As(err, &ce), true)
	assert.Equal(t, ce.Code, http.StatusNotFound)
	assert.Equal(t, ce.Message, "resource not found")
}

func TestCustomError_ErrorsAs_WithWrappedError(t *testing.T) {
	originalErr := BadRequest("bad input")
	wrappedErr := errors.Join(originalErr, errors.New("additional context"))

	var ce *CustomError
	assert.Equal(t, errors.As(wrappedErr, &ce), true)
	assert.Equal(t, ce.Code, http.StatusBadRequest)
}

// CRITICAL TEST: Verify pointer receiver fix for errors.Is()
// This test verifies the fix from CRITICAL_FIXES_APPLIED.md
func TestCustomError_ErrorsIs_PointerReceiverFix(t *testing.T) {
	err1 := NotFound("not found")
	err2 := NotFound("not found")

	// With pointer receiver, errors.Is should work correctly
	// (Before fix: this would return false due to value receiver)
	assert.Equal(t, errors.Is(err1, err2), false) // Different instances, so false is correct

	// But errors.Is with same error should work
	assert.Equal(t, errors.Is(err1, err1), true)

	// And errors.As should extract the correct type
	var ce *CustomError
	assert.Equal(t, errors.As(err1, &ce), true)
	assert.Equal(t, ce.Code, http.StatusNotFound)
}

func TestCustomError_MultipleInstances(t *testing.T) {
	// Create multiple error instances and verify they maintain their identity
	err1 := BadRequest("error 1")
	err2 := BadRequest("error 2")
	err3 := NotFound("error 3")

	// Each should maintain its own message
	assert.Equal(t, err1.Error(), "error 1")
	assert.Equal(t, err2.Error(), "error 2")
	assert.Equal(t, err3.Error(), "error 3")

	// And correct status codes
	assert.Equal(t, GetStatusCode(err1), http.StatusBadRequest)
	assert.Equal(t, GetStatusCode(err2), http.StatusBadRequest)
	assert.Equal(t, GetStatusCode(err3), http.StatusNotFound)
}

func TestCustomError_AsInterface(t *testing.T) {
	// Verify CustomError implements error interface
	var _ error = &CustomError{Code: 500, Message: "test"}
	var _ error = Technical("test")
	var _ error = BadRequest("test")
	var _ error = Unauthorized("test")
	var _ error = Forbidden("test")
	var _ error = NotFound("test")
	var _ error = Conflict("test")
}

// Test deprecated Is() function for backward compatibility
func TestIs_Deprecated(t *testing.T) {
	err1 := NotFound("not found")
	err2 := errors.New("standard error")

	// Should delegate to errors.Is
	assert.Equal(t, Is(err1, err1), true)
	assert.Equal(t, Is(err1, err2), false)
	assert.Equal(t, Is(err2, err2), true)
}

func TestCustomError_EmptyMessage(t *testing.T) {
	err := BadRequest("")
	assert.Equal(t, err.Error(), "")
	assert.Equal(t, GetStatusCode(err), http.StatusBadRequest)
}

func TestCustomError_LongMessage(t *testing.T) {
	longMsg := "This is a very long error message that contains a lot of details about what went wrong in the system and provides context for debugging purposes"
	err := Technical(longMsg)
	assert.Equal(t, err.Error(), longMsg)
	assert.Equal(t, GetStatusCode(err), http.StatusInternalServerError)
}

func TestCustomError_SpecialCharacters(t *testing.T) {
	msg := "Error with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
	err := BadRequest(msg)
	assert.Equal(t, err.Error(), msg)
}
