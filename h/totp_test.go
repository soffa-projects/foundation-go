package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestNewTOPT(t *testing.T) {
	secret, err := NewTOPT("TestIssuer", "test@example.com")

	assert.Equal(t, err, nil)
	assert.NotEqual(t, secret, "")
	assert.Equal(t, len(secret) > 0, true)
}

func TestNewTOPT_WithDifferentIssuer(t *testing.T) {
	secret1, err1 := NewTOPT("Issuer1", "user@example.com")
	secret2, err2 := NewTOPT("Issuer2", "user@example.com")

	assert.Equal(t, err1, nil)
	assert.Equal(t, err2, nil)
	// Secrets should be different
	assert.NotEqual(t, secret1, secret2)
}

func TestNewTOPT_WithDifferentAccount(t *testing.T) {
	secret1, err1 := NewTOPT("MyApp", "user1@example.com")
	secret2, err2 := NewTOPT("MyApp", "user2@example.com")

	assert.Equal(t, err1, nil)
	assert.Equal(t, err2, nil)
	// Secrets should be different
	assert.NotEqual(t, secret1, secret2)
}

func TestNewTOPT_EmptyIssuer(t *testing.T) {
	// Empty issuer is NOT allowed - the library requires it
	_, err := NewTOPT("", "user@example.com")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "Issuer must be set")
}

func TestNewTOPT_EmptyAccount(t *testing.T) {
	// Empty account is NOT allowed - the library requires it
	_, err := NewTOPT("MyApp", "")

	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "AccountName must be set")
}
