package h

import (
	"testing"
)

func TestCheck_WithValidValue(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Check should not panic with valid value, got: %v", r)
		}
	}()
	Check("valid", "should not panic")
	Check(123, "should not panic")
	Check(true, "should not panic")
}

func TestCheck_WithNilValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Check should panic with nil value")
		}
	}()
	Check(nil, "expected panic")
}

func TestCheck_WithNilPointer(t *testing.T) {
	// Note: Check only panics on interface{} == nil, not typed nil pointers
	// This is a limitation of the current implementation
	var ptr *string
	Check(ptr, "does not panic - typed nil is not detected")
}
