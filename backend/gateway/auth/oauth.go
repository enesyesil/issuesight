// Package auth provides authentication functionality for the Gateway service.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// Provider represents an OAuth provider.
type Provider string

const (
	ProviderGitHub Provider = "github"
	ProviderGoogle Provider = "google"
)

// OAuthConfig holds configuration for all OAuth providers.
type OAuthConfig struct {
	GitHub OAuthProviderConfig
	Google OAuthProviderConfig
}

// OAuthProviderConfig holds config for a single OAuth provider.
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// OAuthManager handles OAuth flows for multiple providers.
type OAuthManager struct {
	configs map[Provider]*oauth2.Config
}

// NewOAuthManager creates a new OAuth manager with the given config.
func NewOAuthManager(cfg OAuthConfig, baseURL string) *OAuthManager {
	m := &OAuthManager{
		configs: make(map[Provider]*oauth2.Config),
	}

	// Configure GitHub OAuth
	if cfg.GitHub.ClientID != "" {
		m.configs[ProviderGitHub] = &oauth2.Config{
			ClientID:     cfg.GitHub.ClientID,
			ClientSecret: cfg.GitHub.ClientSecret,
			RedirectURL:  baseURL + "/api/auth/callback",
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		}
	}

	// Configure Google OAuth
	if cfg.Google.ClientID != "" {
		m.configs[ProviderGoogle] = &oauth2.Config{
			ClientID:     cfg.Google.ClientID,
			ClientSecret: cfg.Google.ClientSecret,
			RedirectURL:  baseURL + "/api/auth/callback",
			Scopes:       []string{"email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}

	return m
}

// GetAuthURL returns the OAuth authorization URL for a provider.
func (m *OAuthManager) GetAuthURL(provider Provider, state string) (string, error) {
	cfg, ok := m.configs[provider]
	if !ok {
		return "", fmt.Errorf("oauth: provider %s not configured", provider)
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

// Exchange exchanges an authorization code for an access token.
func (m *OAuthManager) Exchange(ctx context.Context, provider Provider, code string) (*oauth2.Token, error) {
	cfg, ok := m.configs[provider]
	if !ok {
		return nil, fmt.Errorf("oauth: provider %s not configured", provider)
	}
	return cfg.Exchange(ctx, code)
}

// IsConfigured checks if a provider is configured.
func (m *OAuthManager) IsConfigured(provider Provider) bool {
	_, ok := m.configs[provider]
	return ok
}

// OAuthState represents the state parameter for OAuth flow.
type OAuthState struct {
	Provider  Provider  `json:"provider"`
	Nonce     string    `json:"nonce"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GenerateState creates a new OAuth state.
func GenerateState(provider Provider) *OAuthState {
	return &OAuthState{
		Provider:  provider,
		Nonce:     uuid.New().String(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
}

// IsValid checks if the state is still valid.
func (s *OAuthState) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}
