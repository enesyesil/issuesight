package log

import "strings"

// Sensitive key patterns that should be redacted.
//
// WHY REDACT?
// Logs often end up in log aggregators, dashboards, or even terminals
// where others might see them. Accidentally logging a password or API key
// is a security incident.
//
// HOW IT WORKS:
// When logging, we check if the KEY (not value) contains any of these patterns.
// If so, the VALUE is replaced with "[REDACTED]".
//
// Example:
//
//	log.Info("connected", "api_key", "sk-secret123")
//	// Output: level=INFO msg=connected api_key=[REDACTED]
//
// WHY CHECK KEY, NOT VALUE?
// We can't know if a value is sensitive just by looking at it.
// But we CAN know if a key like "password" or "token" likely holds sensitive data.
//
// ADD MORE PATTERNS:
// If your app has other sensitive fields (e.g., "ssn", "credit_card"),
// add them to this list.
var sensitivePatterns = []string{
	"token",
	"password",
	"passwd",
	"secret",
	"key",
	"auth",
	"authorization",
	"credential",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"private",
}

// shouldRedact checks if a key contains sensitive data.
//
// HOW IT WORKS:
//  1. Convert the key to lowercase (case-insensitive matching)
//  2. Check if it contains any sensitive pattern
//  3. Return true if it should be redacted
//
// EXAMPLES:
//
//	shouldRedact("password")     // true
//	shouldRedact("API_KEY")      // true (case insensitive)
//	shouldRedact("github_token") // true (contains "token")
//	shouldRedact("username")     // false
//	shouldRedact("email")        // false
//
// NOTE: This may have false positives.
// "key" matches "keyboard", "monkey", etc.
// In practice, this is rarely a problem for log keys.
func shouldRedact(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// Redact returns "[REDACTED]" if the key is sensitive, otherwise the original value.
//
// WHEN TO USE:
// When you need to explicitly redact a value before logging or displaying.
// Usually you don't need this - the logger's ReplaceAttr handles it automatically.
//
// EXAMPLE:
//
//	fmt.Printf("Config: password=%s\n", log.Redact("password", cfg.Password))
//	// Output: Config: password=[REDACTED]
func Redact(key string, value string) string {
	if shouldRedact(key) {
		return "[REDACTED]"
	}
	return value
}

// RedactMap redacts sensitive values in a map.
//
// WHEN TO USE:
// When you have a map of config or headers and want to safely log it.
//
// EXAMPLE:
//
//	headers := map[string]string{
//	    "Content-Type":  "application/json",
//	    "Authorization": "Bearer sk-secret",
//	}
//	safeHeaders := log.RedactMap(headers)
//	log.Info("request", "headers", safeHeaders)
//	// Output: headers=map[Authorization:[REDACTED] Content-Type:application/json]
//
// NOTE: Returns a NEW map. Original is not modified.
func RedactMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = Redact(k, v)
	}
	return result
}
