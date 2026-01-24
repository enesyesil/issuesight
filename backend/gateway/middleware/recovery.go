// Package middleware provides HTTP middleware for the Gateway service.
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/issuesight/issuesight/internal/platform/log"
)

// Recovery returns a middleware that recovers from panics and logs them.
// Without this, a panic in a handler would crash the entire server.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.Error("panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
					"path", r.URL.Path,
					"method", r.Method,
				)

				// Return 500 Internal Server Error
				http.Error(w, `{"error":"internal_error","message":"Internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
