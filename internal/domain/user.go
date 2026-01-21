package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	DisplayName     string    `json:"display_name"`
	AvatarURL       string    `json:"avatar_url,omitempty"`
	LastRequestedAt time.Time `json:"last_requested_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// UserIdentity links a user to an OAuth provider.
type UserIdentity struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Provider   string // "github" or "google"
	ProviderID string // external user ID
}

// AuthUser is returned after successful authentication.
type AuthUser struct {
	User  *User  `json:"user"`
	Token string `json:"token"` // JWT token
}

// OAuthUserInfo is what we get from OAuth providers.
type OAuthUserInfo struct {
	Provider    string
	ProviderID  string
	Email       string
	DisplayName string
	AvatarURL   string
}
