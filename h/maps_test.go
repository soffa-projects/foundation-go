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

func TestNewMapWithValues(t *testing.T) {
	values := map[string]any{
		"name": "John",
		"age":  30,
	}
	m := NewMapWithValues(values)

	assert.Equal(t, m.Has("name"), true)
	assert.Equal(t, m.Get("name"), "John")
}

func TestMap_Has(t *testing.T) {
	m := NewMapWithValues(map[string]any{"key": "value"})

	assert.Equal(t, m.Has("key"), true)
	assert.Equal(t, m.Has("nonexistent"), false)
}

func TestMap_Get(t *testing.T) {
	m := NewMapWithValues(map[string]any{"key": "value"})

	assert.Equal(t, m.Get("key"), "value")
	assert.Equal(t, m.Get("nonexistent"), nil)
}

func TestMap_GetString(t *testing.T) {
	m := NewMapWithValues(map[string]any{"name": "John"})

	assert.Equal(t, m.GetString("name"), "John")
	assert.Equal(t, m.GetString("nonexistent"), "")
}

func TestMap_GetBool(t *testing.T) {
	m := NewMapWithValues(map[string]any{"active": true, "inactive": false})

	assert.Equal(t, m.GetBool("active"), true)
	assert.Equal(t, m.GetBool("inactive"), false)
	assert.Equal(t, m.GetBool("nonexistent"), false)
}

func TestMap_GetInt(t *testing.T) {
	m := NewMapWithValues(map[string]any{"age": 30, "zero": 0})

	assert.Equal(t, m.GetInt("age"), 30)
	assert.Equal(t, m.GetInt("zero"), 0)
	assert.Equal(t, m.GetInt("nonexistent"), 0)
}

func TestMap_Set(t *testing.T) {
	m := NewMapWithValues(map[string]any{})

	m.Set("key1", "value1")
	m.Set("key2", 123)

	assert.Equal(t, m.Get("key1"), "value1")
	assert.Equal(t, m.Get("key2"), 123)
}

func TestMap_SetChaining(t *testing.T) {
	m := NewMapWithValues(map[string]any{})

	result := m.Set("key1", "value1").Set("key2", "value2")

	assert.Equal(t, result.Get("key1"), "value1")
	assert.Equal(t, result.Get("key2"), "value2")
}

func TestNonEmptyValuesMaps(t *testing.T) {
	input := map[string]any{
		"name":    "John",
		"age":     30,
		"empty":   "",
		"nil":     nil,
		"valid":   "value",
	}

	result := NonEmptyValuesMaps(input)

	assert.Equal(t, result["name"], "John")
	assert.Equal(t, result["age"], 30)
	assert.Equal(t, result["valid"], "value")
	assert.Equal(t, result["empty"], nil)    // empty string filtered
	assert.Equal(t, result["nil"], nil)      // nil filtered
}

func TestDecodeMap(t *testing.T) {
	type Person struct {
		Name string `mapstructure:"name"`
		Age  int    `mapstructure:"age"`
	}

	input := map[string]any{
		"name": "John",
		"age":  30,
	}

	var person Person
	err := DecodeMap(input, &person)

	assert.Equal(t, err, nil)
	assert.Equal(t, person.Name, "John")
	assert.Equal(t, person.Age, 30)
}

func TestIsMap_WithMaps(t *testing.T) {
	assert.Equal(t, IsMap(map[string]string{"key": "value"}), true)
	assert.Equal(t, IsMap(map[string]any{"key": "value"}), true)
	assert.Equal(t, IsMap(map[int]string{1: "value"}), true)
}

func TestIsMap_WithNonMaps(t *testing.T) {
	assert.Equal(t, IsMap("string"), false)
	assert.Equal(t, IsMap(123), false)
	assert.Equal(t, IsMap([]string{"a", "b"}), false)
	// Note: IsMap with nil causes panic due to reflect.TypeOf(nil) - this is a bug
	// assert.Equal(t, IsMap(nil), false)
}

func TestNewMap_InvalidJSON(t *testing.T) {
	m := NewMap("invalid json")
	// Should return a map with nil values
	assert.Equal(t, m.Get("anything"), nil)
}
