package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds configuration for CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is a comma-separated list of allowed origins (e.g., "http://localhost:3000,https://example.com")
	AllowedOrigins string
}

// CORSMiddleware implements configurable CORS handling.
type CORSMiddleware struct {
	allowedOrigins map[string]bool
	allowAll       bool
}

// NewCORS creates a new CORS middleware with default settings.
func NewCORS() *CORSMiddleware {
	return NewCORSWithConfig(CORSConfig{AllowedOrigins: "http://localhost:3000"})
}

// NewCORSWithConfig creates a new CORS middleware with the given config.
func NewCORSWithConfig(cfg CORSConfig) *CORSMiddleware {
	origins := make(map[string]bool)
	allowAll := false

	for _, origin := range strings.Split(cfg.AllowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			allowAll = true
		} else if origin != "" {
			origins[origin] = true
		}
	}

	return &CORSMiddleware{
		allowedOrigins: origins,
		allowAll:       allowAll,
	}
}

// Handler returns the CORS middleware handler.
func (c *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if c.allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && c.allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			// Vary header is required when origin-specific
			w.Header().Add("Vary", "Origin")
		}

		// Allow credentials (cookies) - not compatible with * origin
		if !c.allowAll {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Allow methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		// Allow headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")

		// Cache preflight response for 1 hour
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CORS adds Cross-Origin Resource Sharing headers.
// Deprecated: Use NewCORS().Handler() or NewCORSWithConfig().Handler() for configurable CORS.
func CORS(next http.Handler) http.Handler {
	return NewCORS().Handler(next)
}
