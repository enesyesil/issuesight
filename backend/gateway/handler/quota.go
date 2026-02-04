package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/issuesight/issuesight/backend/gateway/middleware"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/user"
)

// QuotaResponse represents the quota status response.
type QuotaResponse struct {
	Remaining  int       `json:"remaining"`
	Limit      int       `json:"limit"`
	ResetsAt   time.Time `json:"resets_at"`
	IsExceeded bool      `json:"is_exceeded"`
}

// QuotaHandler handles quota-related requests.
type QuotaHandler struct {
	db         *ent.Client
	dailyLimit int
}

// NewQuotaHandler creates a new quota handler.
func NewQuotaHandler(db *ent.Client) *QuotaHandler {
	return &QuotaHandler{
		db:         db,
		dailyLimit: domain.DefaultQuotaLimit,
	}
}

// WithLimit sets the daily limit (0 = unlimited for display purposes).
func (h *QuotaHandler) WithLimit(limit int) *QuotaHandler {
	h.dailyLimit = limit
	return h
}

// Get handles GET /api/quota - returns the user's quota status.
// @Summary      Get Quota
// @Description  Returns the current user's quota status.
// @Tags         quota
// @Produce      json
// @Success      200  {object}  QuotaResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /quota [get]
func (h *QuotaHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get authenticated user
		authUser := middleware.GetUser(ctx)
		if authUser == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		// Get user from database to check last_requested_at
		dbUser, err := h.db.User.Query().
			Where(user.ID(authUser.ID)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				writeError(w, http.StatusUnauthorized, "user_not_found", "User not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch user")
			return
		}

		// Calculate quota status; limit 0 = unlimited (e.g. for testing)
		now := time.Now()
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

		isExceeded := false
		remaining := h.dailyLimit
		if h.dailyLimit <= 0 {
			remaining = -1 // -1 indicates unlimited
		} else if dbUser.LastRequestedAt != nil {
			lastRequest := *dbUser.LastRequestedAt
			if lastRequest.Year() == now.Year() && lastRequest.YearDay() == now.YearDay() {
				isExceeded = true
				remaining = 0
			}
		}

		response := QuotaResponse{
			Remaining:  remaining,
			Limit:      h.dailyLimit,
			ResetsAt:   tomorrow,
			IsExceeded: isExceeded,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
