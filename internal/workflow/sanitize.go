package workflow

import (
	"regexp"
	"strings"
)

var nonAlnumRegex = regexp.MustCompile(`[^a-z0-9()]+`)
var repeatingUnderscoreRegex = regexp.MustCompile(`_{2,}`)

// SanitizeFolderName converts a workflow or credential name into a safe folder name.
// Lowercase, allows parentheses, replaces other non-alphanumeric chars with underscores,
// and trims leading/trailing underscores.
func SanitizeFolderName(name string) string {
	if name == "" {
		return ""
	}

	// Lowercase
	s := strings.ToLower(name)

	// Replace non-alnum (except parentheses) with underscore
	s = nonAlnumRegex.ReplaceAllString(s, "_")

	// Collapse repeating underscores
	s = repeatingUnderscoreRegex.ReplaceAllString(s, "_")

	// Trim leading/trailing underscores
	s = strings.Trim(s, "_")

	return s
}
