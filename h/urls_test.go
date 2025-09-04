package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestEscapeUrl(t *testing.T) {
	uri := "http://localhost:3000/auth/callback"
	expected := "http%3A%2F%2Flocalhost%3A3000%2Fauth%2Fcallback"
	output := EscapeUrl(uri)
	assert.Equal(t, output, expected)
	// double escape should not change the output
	output2 := EscapeUrl(expected)
	assert.Equal(t, output2, expected)
}
