package utils

import (
	"regexp"
	"strings"
)

// CleanMarkdown removes markdown formatting from text
func CleanMarkdown(text string) string {
	// Remove markdown headers (# ## ###)
	text = regexp.MustCompile(`(?m)^#+\s*`).ReplaceAllString(text, "")

	// Remove bold/italic markers (** __ * _)
	text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(text, "$1")

	// Replace multiple newlines with single space
	text = regexp.MustCompile(`\n\n+`).ReplaceAllString(text, " ")

	// Replace single newlines with space
	text = strings.ReplaceAll(text, "\n", " ")

	// Remove extra spaces
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}
