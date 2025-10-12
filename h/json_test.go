package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestToJsonString(t *testing.T) {
	input := map[string]any{
		"login_providers": "google,microsoft",
	}
	output, err := ToJsonString(input)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, output, `{"login_providers":"google,microsoft"}`)
}

func TestFromJsonString(t *testing.T) {
	input := `{"login_providers":"google"}`
	var output map[string]any
	err := FromJsonString(input, &output)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, output, map[string]any{"login_providers": "google"})
}

func TestIsJson_Valid(t *testing.T) {
	assert.Equal(t, IsJson(map[string]any{"key": "value"}), true)
	assert.Equal(t, IsJson([]string{"a", "b"}), true)
	assert.Equal(t, IsJson("string"), true)
	assert.Equal(t, IsJson(123), true)
}

func TestIsJson_Invalid(t *testing.T) {
	// Channels cannot be marshaled to JSON
	ch := make(chan int)
	assert.Equal(t, IsJson(ch), false)
}

func TestNewJsonValue(t *testing.T) {
	json := `{"name":"John","age":30,"active":true}`
	jv := NewJsonValue(json)

	assert.Equal(t, jv.Get("name"), "John")
	assert.Equal(t, jv.Get("age"), float64(30))
	assert.Equal(t, jv.Get("active"), true)
}

func TestNewJsonValue_NestedPath(t *testing.T) {
	json := `{"user":{"name":"John","address":{"city":"NYC"}}}`
	jv := NewJsonValue(json)

	assert.Equal(t, jv.Get("user.name"), "John")
	assert.Equal(t, jv.Get("user.address.city"), "NYC")
}

func TestNewJsonValue_NonExistent(t *testing.T) {
	json := `{"name":"John"}`
	jv := NewJsonValue(json)

	assert.Equal(t, jv.Get("nonexistent"), nil)
}

func TestNewJsonValue_Array(t *testing.T) {
	json := `{"items":["a","b","c"]}`
	jv := NewJsonValue(json)

	items := jv.Get("items")
	assert.NotEqual(t, items, nil)
}

func TestToJsonString_Error(t *testing.T) {
	// Channels cannot be marshaled to JSON
	ch := make(chan int)
	_, err := ToJsonString(ch)
	assert.NotEqual(t, err, nil)
}

func TestFromJsonString_Error(t *testing.T) {
	var output map[string]any
	err := FromJsonString("invalid json", &output)
	assert.NotEqual(t, err, nil)
}
