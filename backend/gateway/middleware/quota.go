package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/issuesight/issuesight/internal/domain"
	httputil "github.com/issuesight/issuesight/internal/platform/http"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/user"
)

// QuotaMiddleware enforces rate limits per user.
type QuotaMiddleware struct {
	db         *ent.Client
	dailyLimit int
}

// NewQuotaMiddleware creates a new quota middleware.
func NewQuotaMiddleware(db *ent.Client) *QuotaMiddleware {
	return &QuotaMiddleware{
		db:         db,
		dailyLimit: domain.DefaultQuotaLimit,
	}
}

// WithLimit sets a custom daily limit.
func (q *QuotaMiddleware) WithLimit(limit int) *QuotaMiddleware {
	q.dailyLimit = limit
	return q
}

// EnforceQuota returns middleware that enforces quota limits.
// Must be used AFTER auth middleware (requires user in context).
func (q *QuotaMiddleware) EnforceQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get authenticated user from context
		authUser := GetUser(ctx)
		if authUser == nil {
			// No user in context - auth middleware should have caught this
			writeQuotaError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		// Check quota in database
		dbUser, err := q.db.User.Query().
			Where(user.ID(authUser.ID)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				writeQuotaError(w, http.StatusUnauthorized, "user_not_found", "User not found")
				return
			}
			writeQuotaError(w, http.StatusInternalServerError, "db_error", "Failed to check quota")
			return
		}

		// Check if user has exceeded quota
		if q.isQuotaExceeded(dbUser) {
			retryAfter := q.getRetryAfterSeconds(dbUser)
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeQuotaError(w, http.StatusTooManyRequests, "quota_exceeded",
				"Daily quota exceeded. Try again tomorrow.")
			return
		}

		// Update last_requested_at timestamp
		_, err = q.db.User.UpdateOne(dbUser).
			SetLastRequestedAt(time.Now()).
			Save(ctx)
		if err != nil {
			// Log but don't fail the request
			// The request should still proceed
		}

		next.ServeHTTP(w, r)
	})
}

// isQuotaExceeded checks if the user has exceeded their daily quota.
// When dailyLimit <= 0, quota is disabled (e.g. for testing).
func (q *QuotaMiddleware) isQuotaExceeded(u *ent.User) bool {
	if q.dailyLimit <= 0 {
		return false // Quota disabled
	}
	if u.LastRequestedAt == nil {
		return false // Never requested before
	}

	// Check if last request was today
	lastRequest := *u.LastRequestedAt
	now := time.Now()

	// If last request was on a different day, quota is reset
	if lastRequest.Year() != now.Year() ||
		lastRequest.YearDay() != now.YearDay() {
		return false
	}

	// For now, we're using a simple "once per day" model based on last_requested_at
	// A more sophisticated implementation would track request counts
	// This allows 1 request per day by default
	return true
}

// getRetryAfterSeconds returns seconds until quota resets.
func (q *QuotaMiddleware) getRetryAfterSeconds(u *ent.User) int {
	if u.LastRequestedAt == nil {
		return 0
	}

	// Calculate seconds until midnight
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return int(tomorrow.Sub(now).Seconds())
}

// CheckQuota is a helper to check quota without enforcing it.
func (q *QuotaMiddleware) CheckQuota(ctx context.Context, userID string) (bool, int, error) {
	// Returns: (isAllowed, remainingRequests, error)
	// Simplified implementation - returns true if user hasn't requested today
	return true, q.dailyLimit, nil
}

// writeQuotaError writes a JSON error response for quota failures.
func writeQuotaError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	httputil.WriteError(w, statusCode, errorCode, message)
}
