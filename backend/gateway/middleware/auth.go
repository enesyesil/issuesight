package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	httputil "github.com/issuesight/issuesight/internal/platform/http"
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

const authTokenCookieName = "auth_token"

// NewAuthMiddleware creates a new auth middleware.
func NewAuthMiddleware(config JWTConfig) *AuthMiddleware {
	return &AuthMiddleware{config: config}
}

// RequireAuth returns middleware that requires a valid JWT token.
func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := a.extractToken(r)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "invalid_token", err.Error())
			return
		}
		if tokenString == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing_token", "Authentication token is required")
			return
		}

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
		tokenString, err := a.extractToken(r)
		if err == nil && tokenString != "" {
			if user, err := a.validateToken(tokenString); err == nil {
				ctx := context.WithValue(r.Context(), UserKey{}, user)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// JWTClaims represents the claims in our JWT tokens.
type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// validateToken validates a JWT token and returns the user.
func (a *AuthMiddleware) validateToken(tokenString string) (*AuthUser, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.config.Secret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if token is expired (jwt library checks this, but be explicit)
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	return &AuthUser{
		ID:    claims.UserID,
		Email: claims.Email,
	}, nil
}

func (a *AuthMiddleware) extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return "", ErrInvalidToken
		}
		return parts[1], nil
	}

	cookie, err := r.Cookie(authTokenCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return "", ErrInvalidToken
	}

	return "", nil
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
	httputil.WriteError(w, statusCode, errorCode, message)
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
