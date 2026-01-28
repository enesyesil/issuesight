package middleware

import (
	"net/http"
)

// SecurityHeaders adds common security headers to every response (OWASP).
// https://cheatsheetseries.owasp.org/cheatsheets/HTTP_Headers_Cheat_Sheet.html
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent Clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Content Security Policy (Basic default, restrictive)
		// Allows scripts/styles from self. Adjust as needed for external assets (e.g. Google Fonts/Analytics).
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self'")

		// HSTS (HTTP Strict Transport Security) - 1 year
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}
