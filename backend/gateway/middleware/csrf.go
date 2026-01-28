package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

// CSRF implements Double Submit Cookie pattern for CSRF protection.
type CSRF struct {
	cookieName  string
	headerName  string
	safeMethods map[string]bool
}

// NewCSRF creates a new CSRF middleware.
func NewCSRF() *CSRF {
	return &CSRF{
		cookieName: "csrf_token",
		headerName: "X-CSRF-Token",
		safeMethods: map[string]bool{
			"GET":     true,
			"HEAD":    true,
			"OPTIONS": true,
			"TRACE":   true,
		},
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
				// Secure:   true, // TODO: Enable in production
			})
		} else {
			token = cookie.Value
		}

		// 2. Validate token on unsafe methods
		if !c.safeMethods[r.Method] {
			requestToken := r.Header.Get(c.headerName)
			if requestToken == "" {
				http.Error(w, "Missing CSRF token", http.StatusForbidden)
				return
			}

			// Simple string comparison (Double Submit Cookie)
			// For higher security, use HMAC binding to session
			if requestToken != token {
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func generateRandomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
