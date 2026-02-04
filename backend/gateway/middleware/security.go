package middleware

import (
	"net/http"
)

// SecurityConfig holds configuration for security middleware.
type SecurityConfig struct {
	// EnableHSTS enables HTTP Strict Transport Security header (should be true in production)
	EnableHSTS bool
}

// SecurityHeadersMiddleware is a configurable security headers middleware.
type SecurityHeadersMiddleware struct {
	enableHSTS bool
}

// NewSecurityHeaders creates a new security headers middleware with default settings.
func NewSecurityHeaders() *SecurityHeadersMiddleware {
	return NewSecurityHeadersWithConfig(SecurityConfig{EnableHSTS: false})
}

// NewSecurityHeadersWithConfig creates a new security headers middleware with the given config.
func NewSecurityHeadersWithConfig(cfg SecurityConfig) *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{
		enableHSTS: cfg.EnableHSTS,
	}
}

// Handler returns the middleware handler.
// https://cheatsheetseries.owasp.org/cheatsheets/HTTP_Headers_Cheat_Sheet.html
func (s *SecurityHeadersMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent Clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Content Security Policy (Basic default, restrictive)
		// Allows scripts/styles from self. Adjust as needed for external assets (e.g. Google Fonts/Analytics).
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self'")

		// HSTS (HTTP Strict Transport Security) - 1 year
		// Only enable in production with HTTPS
		if s.enableHSTS {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders adds common security headers to every response (OWASP).
// Deprecated: Use NewSecurityHeaders().Handler() for configurable security headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return NewSecurityHeaders().Handler(next)
}
