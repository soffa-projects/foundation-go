package h

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestTextToHTML_Simple(t *testing.T) {
	input := "Hello World"
	expected := "<div>Hello World</div>"
	result := TextToHTML(input)
	assert.Equal(t, result, expected)
}

func TestTextToHTML_WithSingleNewline(t *testing.T) {
	input := "Line 1\nLine 2"
	expected := "<div>Line 1<br>Line 2</div>"
	result := TextToHTML(input)
	assert.Equal(t, result, expected)
}

func TestTextToHTML_WithMultipleNewlines(t *testing.T) {
	input := "Line 1\n\nLine 2\n\n\nLine 3"
	expected := "<div>Line 1<br>Line 2<br>Line 3</div>"
	result := TextToHTML(input)
	assert.Equal(t, result, expected)
}

func TestTextToHTML_WithHTMLChars(t *testing.T) {
	input := "<script>alert('xss')</script>"
	result := TextToHTML(input)
	// Should escape HTML
	assert.Equal(t, result, "<div>&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;</div>")
}

func TestTextToHTML_WithAmpersand(t *testing.T) {
	input := "Tom & Jerry"
	expected := "<div>Tom &amp; Jerry</div>"
	result := TextToHTML(input)
	assert.Equal(t, result, expected)
}

func TestTextToHTML_WithQuotes(t *testing.T) {
	input := `He said "Hello"`
	result := TextToHTML(input)
	assert.Equal(t, result, "<div>He said &#34;Hello&#34;</div>")
}

func TestTextToHTML_Empty(t *testing.T) {
	input := ""
	expected := "<div></div>"
	result := TextToHTML(input)
	assert.Equal(t, result, expected)
}
