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
