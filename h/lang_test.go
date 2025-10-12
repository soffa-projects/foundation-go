package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestToInt_Valid(t *testing.T) {
	assert.Equal(t, ToInt("123"), 123)
	assert.Equal(t, ToInt("0"), 0)
	assert.Equal(t, ToInt("-456"), -456)
	assert.Equal(t, ToInt("999999"), 999999)
}

func TestToInt_Invalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("ToInt should panic with invalid input")
		}
	}()
	ToInt("not-a-number")
}

func TestToInt_Empty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("ToInt should panic with empty input")
		}
	}()
	ToInt("")
}

func TestToInt_Float(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("ToInt should panic with float input")
		}
	}()
	ToInt("123.45")
}
