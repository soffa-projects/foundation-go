package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestIsProduction_WithProd(t *testing.T) {
	assert.Equal(t, IsProduction("prod"), true)
	assert.Equal(t, IsProduction("production"), true)
	assert.Equal(t, IsProduction("PROD"), true)
	assert.Equal(t, IsProduction("PRODUCTION"), true)
	assert.Equal(t, IsProduction("Prod"), true)
}

func TestIsProduction_WithNonProd(t *testing.T) {
	assert.Equal(t, IsProduction("dev"), false)
	assert.Equal(t, IsProduction("development"), false)
	assert.Equal(t, IsProduction("test"), false)
	assert.Equal(t, IsProduction("staging"), false)
	assert.Equal(t, IsProduction(""), false)
}
