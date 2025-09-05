package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestMap(t *testing.T) {
	input := "{\"login_providers\":\"google\"}"
	m := NewMap(input)
	assert.Equal(t, m.GetString("login_providers"), "google")
}
