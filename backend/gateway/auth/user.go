package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/user"
	"github.com/issuesight/issuesight/internal/platform/db/ent/useridentity"
)

// UserInfo represents user information from OAuth providers.
type UserInfo struct {
	Provider    Provider
	ProviderID  string
	Email       string
	DisplayName string
	AvatarURL   string
}

// UserManager handles user lookup and creation.
type UserManager struct {
	db *ent.Client
}

// NewUserManager creates a new user manager.
func NewUserManager(db *ent.Client) *UserManager {
	return &UserManager{db: db}
}

// FindOrCreateUser finds an existing user or creates a new one.
func (m *UserManager) FindOrCreateUser(ctx context.Context, info *UserInfo) (*ent.User, error) {
	// First, try to find by provider identity
	identity, err := m.db.UserIdentity.Query().
		Where(
			useridentity.Provider(string(info.Provider)),
			useridentity.ProviderID(info.ProviderID),
		).
		WithUser().
		First(ctx)

	if err == nil && identity.Edges.User != nil {
		// Found existing user with this identity
		return identity.Edges.User, nil
	}

	if !ent.IsNotFound(err) && err != nil {
		return nil, err
	}

	// Identity not found - check if user exists by email
	existingUser, err := m.db.User.Query().
		Where(user.Email(info.Email)).
		First(ctx)

	if err == nil {
		// User exists with same email - link the new identity
		_, err = m.db.UserIdentity.Create().
			SetUserID(existingUser.ID).
			SetProvider(string(info.Provider)).
			SetProviderID(info.ProviderID).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		return existingUser, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	// Create new user and identity in a transaction
	tx, err := m.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	newUser, err := tx.User.Create().
		SetEmail(info.Email).
		SetDisplayName(info.DisplayName).
		SetAvatarURL(info.AvatarURL).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.UserIdentity.Create().
		SetUserID(newUser.ID).
		SetProvider(string(info.Provider)).
		SetProviderID(info.ProviderID).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return newUser, nil
}

// GetUserByID retrieves a user by their ID.
func (m *UserManager) GetUserByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	return m.db.User.Get(ctx, id)
}

// GetUserByEmail retrieves a user by their email.
func (m *UserManager) GetUserByEmail(ctx context.Context, email string) (*ent.User, error) {
	return m.db.User.Query().
		Where(user.Email(email)).
		First(ctx)
}

// Common errors
var (
	ErrUserNotFound = errors.New("user not found")
)
