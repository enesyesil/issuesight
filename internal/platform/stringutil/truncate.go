// Package stringutil provides shared string utility functions.
package stringutil

// Truncate truncates a string to the specified maximum length.
// If maxLen is less than 4, returns an empty string to avoid panic.
// If the string is longer than maxLen, it adds "..." at the end.
func Truncate(s string, maxLen int) string {
	if maxLen < 4 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// TruncateRunes truncates a string by rune count (Unicode-safe).
// If maxLen is less than 4, returns an empty string.
func TruncateRunes(s string, maxLen int) string {
	if maxLen < 4 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
