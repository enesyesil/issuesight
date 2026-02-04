package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	httputil "github.com/issuesight/issuesight/internal/platform/http"
)

// CSRFConfig holds configuration for CSRF middleware.
type CSRFConfig struct {
	// SecureCookie sets the Secure flag on cookies (should be true in production)
	SecureCookie bool
}

// CSRF implements Double Submit Cookie pattern for CSRF protection.
type CSRF struct {
	cookieName   string
	headerName   string
	safeMethods  map[string]bool
	secureCookie bool
}

// NewCSRF creates a new CSRF middleware with default settings.
func NewCSRF() *CSRF {
	return NewCSRFWithConfig(CSRFConfig{SecureCookie: false})
}

// NewCSRFWithConfig creates a new CSRF middleware with the given config.
func NewCSRFWithConfig(cfg CSRFConfig) *CSRF {
	return &CSRF{
		cookieName: "csrf_token",
		headerName: "X-CSRF-Token",
		safeMethods: map[string]bool{
			"GET":     true,
			"HEAD":    true,
			"OPTIONS": true,
			"TRACE":   true,
		},
		secureCookie: cfg.SecureCookie,
	}
}

// Protect enforces CSRF protection.
func (c *CSRF) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check for existing CSRF cookie
		cookie, err := r.Cookie(c.cookieName)
		var token string

		if err != nil || cookie.Value == "" {
			// Generate new token if missing
			token = generateRandomToken()
			http.SetCookie(w, &http.Cookie{
				Name:     c.cookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: false, // Must be readable by JS
				SameSite: http.SameSiteLaxMode,
				Secure:   c.secureCookie,
			})
		} else {
			token = cookie.Value
		}

		// 2. Validate token on unsafe methods
		if !c.safeMethods[r.Method] {
			requestToken := r.Header.Get(c.headerName)
			if requestToken == "" {
				writeCSRFError(w, http.StatusForbidden, "missing_csrf_token", "CSRF token is required")
				return
			}

			// Simple string comparison (Double Submit Cookie)
			// For higher security, use HMAC binding to session
			if requestToken != token {
				writeCSRFError(w, http.StatusForbidden, "invalid_csrf_token", "CSRF token is invalid")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// writeCSRFError writes a JSON error response for CSRF failures.
func writeCSRFError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	httputil.WriteError(w, statusCode, errorCode, message)
}

func generateRandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// This should never happen with crypto/rand, but if it does,
		// we cannot generate secure tokens
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}
