package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestEscapeUrl(t *testing.T) {
	uri := "http://localhost:3000/auth/callback"
	expected := "http%3A%2F%2Flocalhost%3A3000%2Fauth%2Fcallback"
	output := EscapeUrl(uri)
	assert.Equal(t, output, expected)
	// double escape should not change the output
	output2 := EscapeUrl(expected)
	assert.Equal(t, output2, expected)

	// unescape should not change the output
	output3 := UnescapeUrl(expected)
	assert.Equal(t, output3, uri)
	// double unescape should not change the output
	output4 := UnescapeUrl(output3)
	assert.Equal(t, output4, uri)
}

func TestIsDomainName(t *testing.T) {
	assert.Equal(t, IsDomainName("10.0.0.1"), false)
	assert.Equal(t, IsDomainName("localhost"), true)
	assert.Equal(t, IsDomainName("localhost.com"), true)
	assert.Equal(t, IsDomainName("localhost.com.br"), true)
	assert.Equal(t, IsDomainName("localhost.com.br.br"), true)
	assert.Equal(t, IsDomainName("localhost.com.br.br.br"), true)
	assert.Equal(t, IsDomainName("localhost.com.br.br.br.br"), true)
	assert.Equal(t, IsDomainName("localhost.com.br.br.br.br.br"), true)
}
