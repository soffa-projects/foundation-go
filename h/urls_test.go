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

func TestParseUrl(t *testing.T) {
	parsed, err := ParseUrl("https://user:pass@example.com:8080/path?key=value&foo=bar")

	assert.Equal(t, err, nil)
	assert.Equal(t, parsed.Scheme, "https")
	assert.Equal(t, parsed.Host, "example.com:8080")
	assert.Equal(t, parsed.Path, "/path")
	assert.Equal(t, parsed.User, "user")
	assert.Equal(t, parsed.Password, "pass")
	assert.Equal(t, parsed.HasQueryParam("key"), true)
	assert.Equal(t, parsed.Query("key"), "value")
	assert.Equal(t, parsed.Query("foo"), "bar")
}

func TestParseUrl_Simple(t *testing.T) {
	parsed, err := ParseUrl("http://example.com")

	assert.Equal(t, err, nil)
	assert.Equal(t, parsed.Scheme, "http")
	assert.Equal(t, parsed.Host, "example.com")
	assert.Equal(t, parsed.Path, "")
	assert.Equal(t, parsed.User, "")
	assert.Equal(t, parsed.Password, "")
}

func TestParseUrl_WithoutPassword(t *testing.T) {
	parsed, err := ParseUrl("https://user@example.com/path")

	assert.Equal(t, err, nil)
	assert.Equal(t, parsed.User, "user")
	assert.Equal(t, parsed.Password, "")
}

func TestParseUrl_Invalid(t *testing.T) {
	_, err := ParseUrl("not a url ://invalid")
	assert.NotEqual(t, err, nil)
}

func TestUrl_HasQueryParam(t *testing.T) {
	parsed, _ := ParseUrl("http://example.com?key=value")

	assert.Equal(t, parsed.HasQueryParam("key"), true)
	assert.Equal(t, parsed.HasQueryParam("missing"), false)
}

func TestUrl_QueryWithDefault(t *testing.T) {
	parsed, _ := ParseUrl("http://example.com?key=value")

	assert.Equal(t, parsed.QueryWithDefault("key", "default"), "value")
	assert.Equal(t, parsed.QueryWithDefault("missing", "default"), "default")
}

func TestRemoveParamFromUrl(t *testing.T) {
	result, err := RemoveParamFromUrl("http://example.com?key=value&foo=bar", "key")

	assert.Equal(t, err, nil)
	assert.Equal(t, result, "http://example.com?foo=bar")
}

func TestRemoveParamFromUrl_OnlyParam(t *testing.T) {
	result, err := RemoveParamFromUrl("http://example.com?key=value", "key")

	assert.Equal(t, err, nil)
	assert.Equal(t, result, "http://example.com")
}

func TestRemoveParamFromUrl_NonExistent(t *testing.T) {
	result, err := RemoveParamFromUrl("http://example.com?key=value", "missing")

	assert.Equal(t, err, nil)
	assert.Equal(t, result, "http://example.com?key=value")
}

func TestRemoveParamFromUrl_Invalid(t *testing.T) {
	_, err := RemoveParamFromUrl("://invalid", "key")
	assert.NotEqual(t, err, nil)
}

func TestAppendParamToUrl_NoExisting(t *testing.T) {
	result := AppendParamToUrl("http://example.com", "key", "value")
	assert.Equal(t, result, "http://example.com?key=value")
}

func TestAppendParamToUrl_WithExisting(t *testing.T) {
	result := AppendParamToUrl("http://example.com?foo=bar", "key", "value")
	assert.Equal(t, result, "http://example.com?foo=bar&key=value")
}

func TestAppendParamToUrl_ReplaceExisting(t *testing.T) {
	result := AppendParamToUrl("http://example.com?key=old", "key", "new")
	assert.Equal(t, result, "http://example.com?key=new")
}

func TestAppendParamsToUrl(t *testing.T) {
	params := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}
	result := AppendParamsToUrl("http://example.com", params)

	// All params should be in the URL
	assert.Equal(t, result, "http://example.com?key1=value1&key2=123&key3=true")
}

func TestStripOriginFromUrl(t *testing.T) {
	result, err := StripOriginFromUrl("https://example.com/path?key=value")

	assert.Equal(t, err, nil)
	assert.Equal(t, result, "/path?key=value")
}

func TestStripOriginFromUrl_WithHTMLEntity(t *testing.T) {
	result, err := StripOriginFromUrl("https://example.com/path?key=value&amp;foo=bar")

	assert.Equal(t, err, nil)
	assert.Equal(t, result, "/path?key=value&foo=bar")
}

func TestStripOriginFromUrl_PathOnly(t *testing.T) {
	result, err := StripOriginFromUrl("https://example.com/path")

	assert.Equal(t, err, nil)
	assert.Equal(t, result, "/path")
}

func TestStripOriginFromUrl_Invalid(t *testing.T) {
	_, err := StripOriginFromUrl("://invalid")
	assert.NotEqual(t, err, nil)
}
