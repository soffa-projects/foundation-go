package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestTrimToNull_WithEmptyString(t *testing.T) {
	result := TrimToNull("")
	assert.Equal(t, result, nil)
}

func TestTrimToNull_WithValue(t *testing.T) {
	result := TrimToNull("hello")
	assert.NotEqual(t, result, nil)
	assert.Equal(t, *result, "hello")
}

func TestTrimToEmpty_WithNil(t *testing.T) {
	result := TrimToEmpty(nil)
	assert.Equal(t, result, "")
}

func TestTrimToEmpty_WithValue(t *testing.T) {
	str := "hello"
	result := TrimToEmpty(&str)
	assert.Equal(t, result, "hello")
}

func TestIsEmpty_WithEmptyString(t *testing.T) {
	assert.Equal(t, IsEmpty(""), true)
}

func TestIsEmpty_WithNil(t *testing.T) {
	assert.Equal(t, IsEmpty(nil), true)
}

func TestIsEmpty_WithValue(t *testing.T) {
	assert.Equal(t, IsEmpty("hello"), false)
	assert.Equal(t, IsEmpty(123), false)
	assert.Equal(t, IsEmpty([]string{"a"}), false)
}

func TestIsEmpty_WithEmptySlice(t *testing.T) {
	assert.Equal(t, IsEmpty([]string{}), true)
}

func TestIsEmpty_WithEmptyMap(t *testing.T) {
	assert.Equal(t, IsEmpty(map[string]string{}), true)
}

func TestIsNotEmpty_WithValue(t *testing.T) {
	assert.Equal(t, IsNotEmpty("hello"), true)
	assert.Equal(t, IsNotEmpty(123), true)
}

func TestIsNotEmpty_WithEmpty(t *testing.T) {
	assert.Equal(t, IsNotEmpty(""), false)
	assert.Equal(t, IsNotEmpty(nil), false)
}

func TestStrPtr(t *testing.T) {
	ptr := StrPtr("hello")
	assert.NotEqual(t, ptr, nil)
	assert.Equal(t, *ptr, "hello")
}

func TestPtrStr_WithNil(t *testing.T) {
	result := PtrStr(nil)
	assert.Equal(t, result, "")
}

func TestPtrStr_WithValue(t *testing.T) {
	str := "hello"
	result := PtrStr(&str)
	assert.Equal(t, result, "hello")
}

func TestToMap_Valid(t *testing.T) {
	input := `{"name":"John","age":30}`
	result := ToMap(input)

	assert.NotEqual(t, result, nil)
	assert.Equal(t, result["name"], "John")
	assert.Equal(t, result["age"], float64(30)) // JSON numbers are float64
}

func TestToMap_Invalid(t *testing.T) {
	input := `invalid json`
	result := ToMap(input)

	assert.Equal(t, result, nil)
}

func TestToMap_Empty(t *testing.T) {
	input := `{}`
	result := ToMap(input)

	assert.NotEqual(t, result, nil)
	assert.Equal(t, len(result), 0)
}

func TestStrPtrToLower_WithNil(t *testing.T) {
	result := StrPtrToLower(nil)
	assert.Equal(t, result, nil)
}

func TestStrPtrToLower_WithValue(t *testing.T) {
	str := "HELLO World"
	result := StrPtrToLower(&str)

	assert.NotEqual(t, result, nil)
	assert.Equal(t, *result, "hello world")
}

func TestStrPtrToLower_WithLowercase(t *testing.T) {
	str := "already lowercase"
	result := StrPtrToLower(&str)

	assert.NotEqual(t, result, nil)
	assert.Equal(t, *result, "already lowercase")
}
