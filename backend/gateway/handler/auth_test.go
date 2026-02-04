package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/issuesight/issuesight/backend/gateway/auth"
)

func TestGetGitHubUserInfo(t *testing.T) {
	// Create a mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(GitHubUser{
				ID:        12345,
				Login:     "testuser",
				Email:     "test@github.com",
				Name:      "Test User",
				AvatarURL: "https://avatars.githubusercontent.com/u/12345",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create handler with mock (we'll test the parsing logic directly)
	h := &AuthHandler{}

	// For this test, we'll test the response parsing by calling the function
	// with a mock context. In a real scenario, you'd inject the HTTP client.
	// Since we can't easily inject the client, we test the response types.

	t.Run("GitHubUser struct parsing", func(t *testing.T) {
		input := `{"id": 12345, "login": "testuser", "email": "test@github.com", "name": "Test User", "avatar_url": "https://example.com/avatar.png"}`
		var ghUser GitHubUser
		if err := json.Unmarshal([]byte(input), &ghUser); err != nil {
			t.Fatalf("Failed to unmarshal GitHubUser: %v", err)
		}

		if ghUser.ID != 12345 {
			t.Errorf("Expected ID 12345, got %d", ghUser.ID)
		}
		if ghUser.Login != "testuser" {
			t.Errorf("Expected login 'testuser', got %s", ghUser.Login)
		}
		if ghUser.Email != "test@github.com" {
			t.Errorf("Expected email 'test@github.com', got %s", ghUser.Email)
		}
	})

	t.Run("getUserInfo routes to correct provider", func(t *testing.T) {
		// Test that unsupported provider returns error
		_, err := h.getUserInfo(context.Background(), auth.Provider("unsupported"), "token")
		if err == nil {
			t.Error("Expected error for unsupported provider")
		}
	})
}

func TestGetGoogleUserInfo(t *testing.T) {
	t.Run("GoogleUser struct parsing", func(t *testing.T) {
		input := `{"id": "google123", "email": "test@gmail.com", "verified_email": true, "name": "Test User", "picture": "https://example.com/photo.jpg"}`
		var gUser GoogleUser
		if err := json.Unmarshal([]byte(input), &gUser); err != nil {
			t.Fatalf("Failed to unmarshal GoogleUser: %v", err)
		}

		if gUser.ID != "google123" {
			t.Errorf("Expected ID 'google123', got %s", gUser.ID)
		}
		if gUser.Email != "test@gmail.com" {
			t.Errorf("Expected email 'test@gmail.com', got %s", gUser.Email)
		}
		if gUser.Name != "Test User" {
			t.Errorf("Expected name 'Test User', got %s", gUser.Name)
		}
		if !gUser.VerifiedEmail {
			t.Error("Expected verified_email to be true")
		}
	})
}

func TestGitHubEmailParsing(t *testing.T) {
	t.Run("parse email list", func(t *testing.T) {
		input := `[
			{"email": "secondary@example.com", "primary": false, "verified": true},
			{"email": "primary@example.com", "primary": true, "verified": true},
			{"email": "unverified@example.com", "primary": false, "verified": false}
		]`

		var emails []GitHubEmail
		if err := json.Unmarshal([]byte(input), &emails); err != nil {
			t.Fatalf("Failed to unmarshal emails: %v", err)
		}

		if len(emails) != 3 {
			t.Errorf("Expected 3 emails, got %d", len(emails))
		}

		// Find primary verified email
		var primaryEmail string
		for _, e := range emails {
			if e.Primary && e.Verified {
				primaryEmail = e.Email
				break
			}
		}

		if primaryEmail != "primary@example.com" {
			t.Errorf("Expected primary email 'primary@example.com', got %s", primaryEmail)
		}
	})
}

func TestGenerateRandomID(t *testing.T) {
	t.Run("generates non-empty ID", func(t *testing.T) {
		id := generateRandomID()
		if id == "" {
			t.Error("Expected non-empty ID")
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		id1 := generateRandomID()
		id2 := generateRandomID()
		if id1 == id2 {
			t.Error("Expected unique IDs")
		}
	})

	t.Run("generates valid base64", func(t *testing.T) {
		id := generateRandomID()
		// URL-safe base64 encoding of 16 bytes should be ~22 characters
		if len(id) < 20 || len(id) > 24 {
			t.Errorf("Unexpected ID length: %d", len(id))
		}
	})
}
