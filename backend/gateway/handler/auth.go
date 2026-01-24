package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	states         map[string]*auth.OAuthState // In production, use Redis for state storage
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

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(db *ent.Client) *AuthHandler {
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

	baseURL := "http://localhost:8080" // TODO: Get from config
	if cfg.IsProduction() {
		baseURL = "https://issuesight.com" // TODO: Get from config
	}

	return &AuthHandler{
		db:             db,
		oauthManager:   auth.NewOAuthManager(oauthCfg, baseURL),
		sessionManager: auth.NewSessionManager(cfg.JWTSecret, cfg.JWTExpiration),
		userManager:    auth.NewUserManager(db),
		states:         make(map[string]*auth.OAuthState),
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

		state := h.generateState(auth.ProviderGitHub)
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

		state := h.generateState(auth.ProviderGoogle)
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
// @Success      200    {object}  AuthTokenResponse
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

		// Validate state
		state, ok := h.states[stateParam]
		if !ok || !state.IsValid() {
			writeError(w, http.StatusBadRequest, "invalid_state", "Invalid or expired state")
			return
		}
		delete(h.states, stateParam) // One-time use

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

		// Return token to client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthTokenResponse{
			Token: jwtToken,
			User: AuthUserResponse{
				ID:          user.ID.String(),
				Email:       user.Email,
				DisplayName: user.DisplayName,
				AvatarURL:   user.AvatarURL,
			},
		})
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
		// For JWT-based auth, logout is primarily client-side (discard token)
		// Server can optionally maintain a token blacklist in Redis
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
		// Get authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing_token", "Authorization header required")
			return
		}

		// Extract token
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid authorization header")
			return
		}
		tokenString := authHeader[7:]

		// Validate token
		claims, err := h.sessionManager.ValidateToken(tokenString)
		if err != nil {
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
func (h *AuthHandler) generateState(provider auth.Provider) *auth.OAuthState {
	state := auth.GenerateState(provider)
	h.states[state.Nonce] = state
	return state
}

// getUserInfo fetches user info from the OAuth provider.
func (h *AuthHandler) getUserInfo(ctx context.Context, provider auth.Provider, accessToken string) (*auth.UserInfo, error) {
	// TODO: Implement actual API calls to providers
	// For GitHub: GET https://api.github.com/user with Authorization: token {accessToken}
	// For Google: GET https://www.googleapis.com/oauth2/v2/userinfo with Authorization: Bearer {accessToken}

	// Placeholder - in production, make actual HTTP calls
	return &auth.UserInfo{
		Provider:    provider,
		ProviderID:  generateRandomID(),
		Email:       "user@example.com",
		DisplayName: "Example User",
		AvatarURL:   "",
	}, nil
}

// generateRandomID generates a random ID string.
func generateRandomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
