package middleware

import (
	"net/http"
)

// CORS adds Cross-Origin Resource Sharing headers.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow specific origin (frontend)
		// TODO: Make this configurable via env
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")

		// Allow credentials (cookies)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Allow methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		// Allow headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
