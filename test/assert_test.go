package test

import (
	"errors"
	"testing"
)

func TestNewAssertions(t *testing.T) {
	assert := NewAssertions(t)
	// Should not panic and should create a valid Assertions instance
	if assert.internal == nil {
		t.Error("NewAssertions should create a valid internal gomega instance")
	}
}

func TestAssertions_Nil(t *testing.T) {
	assert := NewAssertions(t)

	// Should not fail with nil error
	assert.Nil(nil)

	// Note: Cannot easily test failure case as it would fail the test
}

func TestAssertions_NotNil_Single(t *testing.T) {
	assert := NewAssertions(t)

	// Should not fail with non-nil value
	value := "test"
	assert.NotNil(value)
}

func TestAssertions_NotNil_Multiple(t *testing.T) {
	assert := NewAssertions(t)

	// Should not fail with multiple non-nil values
	value1 := "test"
	value2 := 42
	value3 := true
	assert.NotNil(value1, value2, value3)
}

func TestAssertions_NotEmpty(t *testing.T) {
	assert := NewAssertions(t)

	// Should not fail with non-empty string
	assert.NotEmpty("test")
}

func TestAssertions_True(t *testing.T) {
	assert := NewAssertions(t)

	// Should not fail with true value
	assert.True(true)
}

func TestAssertions_False(t *testing.T) {
	assert := NewAssertions(t)

	// Should not fail with false value
	assert.False(false)
}

func TestAssertions_Equals(t *testing.T) {
	assert := NewAssertions(t)

	// Test with strings
	assert.Equals("hello", "hello")

	// Test with integers
	assert.Equals(42, 42)

	// Test with booleans
	assert.Equals(true, true)

	// Note: Cannot test nil equality as gomega refuses to compare nil to nil
}

func TestAssertions_MatchJson(t *testing.T) {
	assert := NewAssertions(t)

	// Test with matching JSON
	json1 := `{"name":"John","age":30}`
	json2 := `{"age":30,"name":"John"}` // Different order, same content
	assert.MatchJson(json1, json2)

	// Test with identical JSON
	json3 := `{"key":"value"}`
	assert.MatchJson(json3, json3)
}

func TestAssertions_NotEqual(t *testing.T) {
	assert := NewAssertions(t)

	// Test with different strings
	assert.NotEqual("hello", "world")

	// Test with different integers
	assert.NotEqual(42, 43)

	// Test with different booleans
	assert.NotEqual(true, false)

	// Test nil vs non-nil
	assert.NotEqual(nil, "something")
}

func TestAssertions_NilWithError(t *testing.T) {
	assert := NewAssertions(t)

	// Test that nil error passes
	var err error
	assert.Nil(err)

	// Test with explicitly nil error
	assert.Nil(nil)
}

func TestAssertions_ChainedCalls(t *testing.T) {
	assert := NewAssertions(t)

	// Test multiple assertions in sequence
	assert.NotNil("test")
	assert.True(true)
	assert.False(false)
	assert.Equals(42, 42)
	assert.NotEmpty("non-empty")
}

func TestAssertions_WithComplexTypes(t *testing.T) {
	assert := NewAssertions(t)

	// Test with struct
	type Person struct {
		Name string
		Age  int
	}
	p1 := Person{Name: "John", Age: 30}
	p2 := Person{Name: "John", Age: 30}
	assert.Equals(p1, p2)

	// Test with slice
	slice1 := []int{1, 2, 3}
	slice2 := []int{1, 2, 3}
	assert.Equals(slice1, slice2)

	// Test with map
	map1 := map[string]int{"a": 1, "b": 2}
	map2 := map[string]int{"a": 1, "b": 2}
	assert.Equals(map1, map2)
}

func TestAssertions_WithErrors(t *testing.T) {
	assert := NewAssertions(t)

	// Test NotNil with error
	err := errors.New("test error")
	assert.NotNil(err)

	// Test Equals with same error instance
	err1 := errors.New("test")
	assert.Equals(err1, err1) // Same instance should equal itself
}

func TestAssertions_JsonEdgeCases(t *testing.T) {
	assert := NewAssertions(t)

	// Test with empty JSON object
	assert.MatchJson("{}", "{}")

	// Test with empty JSON array
	assert.MatchJson("[]", "[]")

	// Test with nested JSON
	json1 := `{"outer":{"inner":"value"}}`
	json2 := `{"outer":{"inner":"value"}}`
	assert.MatchJson(json1, json2)

	// Test with JSON array
	json3 := `[1,2,3]`
	json4 := `[1,2,3]`
	assert.MatchJson(json3, json4)
}

func TestAssertions_NotEmptyWithWhitespace(t *testing.T) {
	assert := NewAssertions(t)

	// Whitespace is not empty
	assert.NotEmpty(" ")
	assert.NotEmpty("\t")
	assert.NotEmpty("\n")
}

func TestAssertions_EqualsWithZeroValues(t *testing.T) {
	assert := NewAssertions(t)

	// Test with zero values
	assert.Equals(0, 0)
	assert.Equals("", "")
	assert.Equals(false, false)
}

func TestAssertions_NotNilWithZeroValues(t *testing.T) {
	assert := NewAssertions(t)

	// Zero values are not nil (for value types)
	assert.NotNil(0)
	assert.NotNil("")
	assert.NotNil(false)
}
