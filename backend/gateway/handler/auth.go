package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/issuesight/issuesight/backend/gateway/auth"
	"github.com/issuesight/issuesight/internal/config"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	db             *ent.Client
	oauthManager   *auth.OAuthManager
	sessionManager *auth.SessionManager
	userManager    *auth.UserManager
	stateStore     auth.StateStore
	isProduction   bool
	frontendURL    string
}

// AuthUserResponse represents user details in auth response.
type AuthUserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

// AuthTokenResponse is the response containing the JWT token.
type AuthTokenResponse struct {
	Token string           `json:"token"`
	User  AuthUserResponse `json:"user"`
}

// LogoutResponse represents the logout status.
type LogoutResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// AuthHandlerConfig holds configuration for the auth handler.
type AuthHandlerConfig struct {
	DB         *ent.Client
	StateStore auth.StateStore
}

// NewAuthHandler creates a new auth handler.
// If StateStore is nil, an in-memory store will be used (not recommended for production).
func NewAuthHandler(handlerCfg AuthHandlerConfig) *AuthHandler {
	cfg, _ := config.Load()

	oauthCfg := auth.OAuthConfig{
		GitHub: auth.OAuthProviderConfig{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
		},
		Google: auth.OAuthProviderConfig{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		},
	}

	stateStore := handlerCfg.StateStore
	if stateStore == nil {
		// Fallback to in-memory store (not recommended for production)
		stateStore = auth.NewMemoryStateStore()
	}

	return &AuthHandler{
		db:             handlerCfg.DB,
		oauthManager:   auth.NewOAuthManager(oauthCfg, cfg.BaseURL),
		sessionManager: auth.NewSessionManager(cfg.JWTSecret, cfg.JWTExpiration),
		userManager:    auth.NewUserManager(handlerCfg.DB),
		stateStore:     stateStore,
		isProduction:   cfg.IsProduction(),
		frontendURL:    cfg.FrontendURL,
	}
}

// GitHub initiates GitHub OAuth flow.
// @Summary      Initiate GitHub OAuth
// @Description  Redirects to GitHub for OAuth authentication
// @Tags         auth
// @Success      307
// @Failure      503  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/github [get]
func (h *AuthHandler) GitHub() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.oauthManager.IsConfigured(auth.ProviderGitHub) {
			writeError(w, http.StatusServiceUnavailable, "not_configured", "GitHub OAuth is not configured")
			return
		}

		state, err := h.generateState(r.Context(), auth.ProviderGitHub)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "state_error", "Failed to generate state")
			return
		}

		url, err := h.oauthManager.GetAuthURL(auth.ProviderGitHub, state.Nonce)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "oauth_error", "Failed to generate auth URL")
			return
		}

		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// Google initiates Google OAuth flow.
// @Summary      Initiate Google OAuth
// @Description  Redirects to Google for OAuth authentication
// @Tags         auth
// @Success      307
// @Failure      503  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/google [get]
func (h *AuthHandler) Google() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.oauthManager.IsConfigured(auth.ProviderGoogle) {
			writeError(w, http.StatusServiceUnavailable, "not_configured", "Google OAuth is not configured")
			return
		}

		state, err := h.generateState(r.Context(), auth.ProviderGoogle)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "state_error", "Failed to generate state")
			return
		}

		url, err := h.oauthManager.GetAuthURL(auth.ProviderGoogle, state.Nonce)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "oauth_error", "Failed to generate auth URL")
			return
		}

		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// Callback handles OAuth callback from providers.
// @Summary      OAuth Callback
// @Description  Handles the callback from OAuth providers, creates user session, and returns JWT.
// @Tags         auth
// @Param        state  query     string  true  "OAuth state"
// @Param        code   query     string  true  "OAuth code"
// @Success      200    {object}  AuthUserResponse
// @Failure      400    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /auth/callback [get]
func (h *AuthHandler) Callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get state and code from query params
		stateParam := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")

		if stateParam == "" || code == "" {
			writeError(w, http.StatusBadRequest, "missing_params", "Missing state or code parameter")
			return
		}

		// Validate state (Get also deletes - one-time use)
		state, err := h.stateStore.Get(ctx, stateParam)
		if err != nil || !state.IsValid() {
			writeError(w, http.StatusBadRequest, "invalid_state", "Invalid or expired state")
			return
		}

		// Exchange code for token
		token, err := h.oauthManager.Exchange(ctx, state.Provider, code)
		if err != nil {
			writeError(w, http.StatusBadRequest, "exchange_failed", "Failed to exchange code for token")
			return
		}

		// Get user info from provider
		userInfo, err := h.getUserInfo(ctx, state.Provider, token.AccessToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "user_info_failed", "Failed to get user info")
			return
		}

		// Find or create user
		user, err := h.userManager.FindOrCreateUser(ctx, userInfo)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "user_create_failed", "Failed to create user")
			return
		}

		// Generate JWT token
		jwtToken, err := h.sessionManager.GenerateToken(user.ID, user.Email)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_failed", "Failed to generate token")
			return
		}

		// Set HttpOnly Cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    jwtToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   h.isProduction,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   24 * 60 * 60, // 24 hours
		})

		// Redirect to Frontend Dashboard
		http.Redirect(w, r, h.frontendURL+"/dashboard/issues", http.StatusTemporaryRedirect)
	}
}

// Logout handles user logout.
// @Summary      Logout
// @Description  Logs out the user.
// @Tags         auth
// @Success      200  {object}  LogoutResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Clear the cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LogoutResponse{
			Status:  "logged_out",
			Message: "Successfully logged out",
		})
	}
}

// Me returns information about the currently authenticated user.
// @Summary      Get Current User
// @Description  Returns the profile of the currently authenticated user.
// @Tags         auth
// @Security     BearerAuth
// @Success      200  {object}  AuthUserResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /auth/me [get]
func (h *AuthHandler) Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// 1. Try to get from Cookie
		cookie, err := r.Cookie("auth_token")
		if err == nil {
			tokenString = cookie.Value
		}

		// 2. Fallback to Authorization Header (for API clients/testing)
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString = authHeader[7:]
			}
		}

		if tokenString == "" {
			writeError(w, http.StatusUnauthorized, "missing_token", "Authentication required")
			return
		}

		// Validate token
		claims, err := h.sessionManager.ValidateToken(tokenString)
		if err != nil {
			// If cookie is invalid, clear it
			http.SetCookie(w, &http.Cookie{
				Name:   "auth_token",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			writeError(w, http.StatusUnauthorized, "invalid_token", "Token is invalid or expired")
			return
		}

		// Get user from database
		user, err := h.userManager.GetUserByID(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, http.StatusNotFound, "user_not_found", "User not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthUserResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
		})
	}
}

// generateState creates and stores a new OAuth state.
func (h *AuthHandler) generateState(ctx context.Context, provider auth.Provider) (*auth.OAuthState, error) {
	state := auth.GenerateState(provider)
	if err := h.stateStore.Save(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// getUserInfo fetches user info from the OAuth provider.
func (h *AuthHandler) getUserInfo(ctx context.Context, provider auth.Provider, accessToken string) (*auth.UserInfo, error) {
	switch provider {
	case auth.ProviderGitHub:
		return h.getGitHubUserInfo(ctx, accessToken)
	case auth.ProviderGoogle:
		return h.getGoogleUserInfo(ctx, accessToken)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// GitHubUser represents the user info response from GitHub API.
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// getGitHubUserInfo fetches user info from GitHub API.
func (h *AuthHandler) getGitHubUserInfo(ctx context.Context, accessToken string) (*auth.UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var ghUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("decode github response: %w", err)
	}

	// If email is not public, try to fetch from emails endpoint
	email := ghUser.Email
	if email == "" {
		email, _ = h.getGitHubPrimaryEmail(ctx, accessToken)
	}

	displayName := ghUser.Name
	if displayName == "" {
		displayName = ghUser.Login
	}

	return &auth.UserInfo{
		Provider:    auth.ProviderGitHub,
		ProviderID:  fmt.Sprintf("%d", ghUser.ID),
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   ghUser.AvatarURL,
	}, nil
}

// GitHubEmail represents an email from GitHub's emails endpoint.
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// getGitHubPrimaryEmail fetches the primary email from GitHub's emails endpoint.
func (h *AuthHandler) getGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails api error: status %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// Fallback to any verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}

// GoogleUser represents the user info response from Google API.
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// getGoogleUserInfo fetches user info from Google API.
func (h *AuthHandler) getGoogleUserInfo(ctx context.Context, accessToken string) (*auth.UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("decode google response: %w", err)
	}

	return &auth.UserInfo{
		Provider:    auth.ProviderGoogle,
		ProviderID:  gUser.ID,
		Email:       gUser.Email,
		DisplayName: gUser.Name,
		AvatarURL:   gUser.Picture,
	}, nil
}

// generateRandomID generates a random ID string.
func generateRandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// This should never happen with crypto/rand, but if it does,
		// we cannot generate secure tokens
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}
