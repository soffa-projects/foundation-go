package micro

import (
	"html"
	"regexp"
	"strings"
)

func TextToHTML(text string) string {
	// Escape HTML special characters to prevent XSS
	escaped := html.EscapeString(text)

	// Replace multiple consecutive newlines with a single <br> tag
	// This regex matches 2 or more consecutive newlines and replaces them with a single newline
	multipleNewlines := regexp.MustCompile(`\n{2,}`)
	withSingleNewlines := multipleNewlines.ReplaceAllString(escaped, "\n")

	// Replace remaining single newlines with <br> tags
	withLineBreaks := strings.ReplaceAll(withSingleNewlines, "\n", "<br>")

	// Wrap in a div for proper HTML structure
	return "<div>" + withLineBreaks + "</div>"
}
