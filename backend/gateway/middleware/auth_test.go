package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-key-for-testing-only"

func createTestToken(userID uuid.UUID, email string, secret string, expiration time.Duration) string {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestValidateToken_ValidToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	userID := uuid.New()
	email := "test@example.com"
	tokenString := createTestToken(userID, email, testSecret, time.Hour)

	user, err := auth.validateToken(tokenString)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if user.ID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, user.ID)
	}

	if user.Email != email {
		t.Errorf("Expected email %s, got %s", email, user.Email)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	userID := uuid.New()
	// Create a token that expired 1 hour ago
	tokenString := createTestToken(userID, "test@example.com", testSecret, -time.Hour)

	_, err := auth.validateToken(tokenString)
	if err == nil {
		t.Fatal("Expected error for expired token, got nil")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	userID := uuid.New()
	// Create token with different secret
	tokenString := createTestToken(userID, "test@example.com", "wrong-secret", time.Hour)

	_, err := auth.validateToken(tokenString)
	if err == nil {
		t.Fatal("Expected error for invalid signature, got nil")
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	testCases := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"garbage", "not-a-valid-jwt"},
		{"partial jwt", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"random base64", "aGVsbG8gd29ybGQ=.aGVsbG8gd29ybGQ=.aGVsbG8gd29ybGQ="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := auth.validateToken(tc.token)
			if err == nil {
				t.Errorf("Expected error for token %q, got nil", tc.token)
			}
		})
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	userID := uuid.New()
	email := "test@example.com"
	tokenString := createTestToken(userID, email, testSecret, time.Hour)

	// Create a test handler that checks for user in context
	handler := auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user == nil {
			t.Error("Expected user in context, got nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if user.ID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, user.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequireAuth_ValidTokenCookie(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	userID := uuid.New()
	email := "test@example.com"
	tokenString := createTestToken(userID, email, testSecret, time.Hour)

	handler := auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user == nil {
			t.Error("Expected user in context, got nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if user.ID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, user.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenString})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequireAuth_MissingToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	handler := auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without token")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	handler := auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid token")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestOptionalAuth_WithToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	userID := uuid.New()
	tokenString := createTestToken(userID, "test@example.com", testSecret, time.Hour)

	handler := auth.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user == nil {
			t.Error("Expected user in context when token provided")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestOptionalAuth_WithCookie(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	userID := uuid.New()
	tokenString := createTestToken(userID, "test@example.com", testSecret, time.Hour)

	handler := auth.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user == nil {
			t.Error("Expected user in context when cookie provided")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/optional", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenString})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestOptionalAuth_WithoutToken(t *testing.T) {
	auth := NewAuthMiddleware(JWTConfig{
		Secret:     testSecret,
		Expiration: time.Hour,
	})

	handler := auth.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user != nil {
			t.Error("Expected no user in context when no token provided")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/optional", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}
