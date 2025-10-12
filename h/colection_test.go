package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestEmptyIfNull_WithNil(t *testing.T) {
	var nilSlice []string
	result := EmptyIfNull(nilSlice)
	assert.Equal(t, len(result), 0)
	assert.NotEqual(t, result, nil)
}

func TestEmptyIfNull_WithEmptySlice(t *testing.T) {
	emptySlice := []string{}
	result := EmptyIfNull(emptySlice)
	assert.Equal(t, len(result), 0)
}

func TestEmptyIfNull_WithValues(t *testing.T) {
	slice := []string{"a", "b", "c"}
	result := EmptyIfNull(slice)
	assert.Equal(t, len(result), 3)
	assert.Equal(t, result, slice)
}

func TestContainsString_WithMatch(t *testing.T) {
	array := []string{"apple", "banana", "orange"}
	assert.Equal(t, ContainsString(array, "banana"), true)
	assert.Equal(t, ContainsString(array, "apple"), true)
	assert.Equal(t, ContainsString(array, "orange"), true)
}

func TestContainsString_WithoutMatch(t *testing.T) {
	array := []string{"apple", "banana", "orange"}
	assert.Equal(t, ContainsString(array, "grape"), false)
	assert.Equal(t, ContainsString(array, ""), false)
}

func TestContainsString_EmptyArray(t *testing.T) {
	assert.Equal(t, ContainsString([]string{}, "anything"), false)
}

func TestContainsString_EmptyValue(t *testing.T) {
	array := []string{"apple", "banana"}
	assert.Equal(t, ContainsString(array, ""), false)
}

func TestContainsAnyString_WithMatch(t *testing.T) {
	array := []string{"apple", "banana", "orange"}
	assert.Equal(t, ContainsAnyString(array, []string{"grape", "banana"}), true)
	assert.Equal(t, ContainsAnyString(array, []string{"apple"}), true)
}

func TestContainsAnyString_WithoutMatch(t *testing.T) {
	array := []string{"apple", "banana", "orange"}
	assert.Equal(t, ContainsAnyString(array, []string{"grape", "mango"}), false)
}

func TestContainsAnyString_EmptyArray(t *testing.T) {
	assert.Equal(t, ContainsAnyString([]string{}, []string{"anything"}), false)
}

func TestContainsAnyString_EmptyValues(t *testing.T) {
	array := []string{"apple", "banana"}
	assert.Equal(t, ContainsAnyString(array, []string{}), false)
}
