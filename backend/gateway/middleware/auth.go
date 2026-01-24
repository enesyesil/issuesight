package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserKey is the context key for authenticated user info.
type UserKey struct{}

// AuthUser represents the authenticated user in the request context.
type AuthUser struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// AuthMiddleware creates authentication middleware with the given config.
type AuthMiddleware struct {
	config JWTConfig
}

// NewAuthMiddleware creates a new auth middleware.
func NewAuthMiddleware(config JWTConfig) *AuthMiddleware {
	return &AuthMiddleware{config: config}
}

// RequireAuth returns middleware that requires a valid JWT token.
func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing_token", "Authorization header is required")
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeAuthError(w, http.StatusUnauthorized, "invalid_token", "Invalid authorization header format")
			return
		}

		tokenString := parts[1]

		// Validate JWT token
		user, err := a.validateToken(tokenString)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "invalid_token", "Token is invalid or expired")
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth returns middleware that extracts user if token present, but doesn't require it.
func (a *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				if user, err := a.validateToken(parts[1]); err == nil {
					ctx := context.WithValue(r.Context(), UserKey{}, user)
					r = r.WithContext(ctx)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// validateToken validates a JWT token and returns the user.
// This is a simplified implementation - in production, use a proper JWT library.
func (a *AuthMiddleware) validateToken(tokenString string) (*AuthUser, error) {
	// TODO: Implement proper JWT validation using a library like golang-jwt
	// For now, this is a placeholder that accepts any non-empty token
	// In production:
	// 1. Parse the JWT token
	// 2. Verify the signature using a.config.Secret
	// 3. Check expiration
	// 4. Extract user claims

	// Placeholder - return error for empty tokens
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// This should be replaced with actual JWT parsing
	return nil, ErrInvalidToken
}

// GetUser retrieves the authenticated user from context.
func GetUser(ctx context.Context) *AuthUser {
	if user, ok := ctx.Value(UserKey{}).(*AuthUser); ok {
		return user
	}
	return nil
}

// writeAuthError writes a JSON error response for auth failures.
func writeAuthError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   errorCode,
		"message": message,
	})
}

// Sentinel errors for auth.
var (
	ErrInvalidToken = &AuthError{Message: "invalid or expired token"}
	ErrMissingToken = &AuthError{Message: "missing authorization token"}
)

// AuthError represents an authentication error.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
