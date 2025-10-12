package h

import (
	"errors"
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestSafe_WithNoError(t *testing.T) {
	result := Safe("value", nil)
	assert.Equal(t, result, "value")

	result2 := Safe(123, nil)
	assert.Equal(t, result2, 123)

	result3 := Safe(true, nil)
	assert.Equal(t, result3, true)
}

func TestSafe_WithError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Safe should panic when error is provided")
		}
	}()
	Safe("value", errors.New("test error"))
}

func TestIsSameFunc_SameFunction(t *testing.T) {
	fn1 := func() {}
	fn2 := fn1
	assert.Equal(t, IsSameFunc(fn1, fn2), true)
}

func TestIsSameFunc_DifferentFunctions(t *testing.T) {
	fn1 := func() {}
	fn2 := func() {}
	assert.Equal(t, IsSameFunc(fn1, fn2), false)
}

func TestIsSameFunc_NamedFunctions(t *testing.T) {
	// Compare with itself
	assert.Equal(t, IsSameFunc(TestSafe_WithNoError, TestSafe_WithNoError), true)

	// Compare different functions
	assert.Equal(t, IsSameFunc(TestSafe_WithNoError, TestSafe_WithError), false)
}
