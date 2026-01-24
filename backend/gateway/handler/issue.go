package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/issuesight/issuesight/backend/collector/parser"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/cache"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/lock"
	"github.com/issuesight/issuesight/internal/platform/stream"
)

// IssueSubmitRequest is the request body for submitting an issue.
type IssueSubmitRequest struct {
	URL string `json:"url"`
}

// IssueSubmitResponse is the response for a submitted issue.
type IssueSubmitResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// IssueHandler handles issue submission requests.
type IssueHandler struct {
	publisher stream.Publisher
	locker    lock.Locker
	cache     cache.Cache
	db        *ent.Client
}

// NewIssueHandler creates a new issue handler.
func NewIssueHandler(publisher stream.Publisher, locker lock.Locker, cache cache.Cache, db *ent.Client) *IssueHandler {
	return &IssueHandler{
		publisher: publisher,
		locker:    locker,
		cache:     cache,
		db:        db,
	}
}

// Submit handles POST /api/issues - submitting an issue URL for processing.
// @Summary      Submit Issue
// @Description  Submits a GitHub issue URL for processing.
// @Tags         issues
// @Accept       json
// @Produce      json
// @Param        request body      IssueSubmitRequest  true  "Issue submission request"
// @Success      202     {object}  IssueSubmitResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      409     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /issues [post]
func (h *IssueHandler) Submit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request body
		var req IssueSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
			return
		}

		// Parse and validate the GitHub issue URL
		issueURL, err := parser.Parse(req.URL)
		if err != nil {
			var errMsg string
			switch {
			case errors.Is(err, parser.ErrEmptyURL):
				errMsg = "URL is required"
			case errors.Is(err, parser.ErrInvalidURL):
				errMsg = "Invalid URL format"
			case errors.Is(err, parser.ErrInvalidHost):
				errMsg = "URL must be from github.com"
			case errors.Is(err, parser.ErrPullRequestURL):
				errMsg = "Pull request URLs are not supported"
			case errors.Is(err, parser.ErrInvalidPath):
				errMsg = "Invalid GitHub issue URL path"
			case errors.Is(err, parser.ErrInvalidIssueNumber):
				errMsg = "Invalid issue number"
			default:
				errMsg = "Failed to parse URL"
			}
			writeError(w, http.StatusBadRequest, "invalid_url", errMsg)
			return
		}

		// Generate unique key for this issue using fmt.Sprintf for proper int formatting
		issueKey := fmt.Sprintf("%s/%s/%d", issueURL.Owner, issueURL.Repo, issueURL.Number)
		lockKey := domain.LockKeyIssue + issueKey
		cacheKey := domain.CacheKeyIssue + issueKey

		// Try to acquire distributed lock to prevent duplicate processing
		issueLock, err := h.locker.TryAcquire(ctx, lockKey, domain.DefaultLockTTL)
		if err != nil {
			if errors.Is(err, lock.ErrLockNotAcquired) {
				writeError(w, http.StatusConflict, "already_processing", "This issue is already being processed")
				return
			}
			writeError(w, http.StatusInternalServerError, "lock_error", "Failed to acquire lock")
			return
		}
		defer issueLock.Release(ctx)

		// Check if issue was already processed (check cache first)
		_, err = h.cache.Get(ctx, cacheKey)
		if err == nil {
			// Found in cache - already processed or processing
			writeError(w, http.StatusConflict, "already_processed", "This issue has already been processed")
			return
		}
		// Cache miss is expected, continue

		// Publish to Redis Stream for processing
		payload := map[string]interface{}{
			"owner":        issueURL.Owner,
			"repo":         issueURL.Repo,
			"issue_number": issueURL.Number,
			"submitted_at": time.Now().UTC().Format(time.RFC3339),
		}

		_, err = h.publisher.Publish(ctx, domain.StreamGitHubEvents, payload)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "publish_error", "Failed to queue issue for processing")
			return
		}

		// Mark as processing in cache to prevent duplicates
		_ = h.cache.Set(ctx, cacheKey, []byte("processing"), domain.DefaultCacheTTL)

		// Return success response
		response := IssueSubmitResponse{
			Status:  "queued",
			Message: "Issue queued for processing",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}
