package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/issuesight/issuesight/backend/collector/parser"
	"github.com/issuesight/issuesight/backend/gateway/middleware"
	"github.com/issuesight/issuesight/internal/concepts"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/cache"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/concept"
	"github.com/issuesight/issuesight/internal/platform/log"
)

// maxRequestBodySize limits request body to 1MB to prevent DoS attacks.
const maxRequestBodySize = 1 << 20 // 1MB

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
	collectorURL string
	httpClient   *http.Client
	cache        cache.Cache
	db           *ent.Client
}

// NewIssueHandler creates a new issue handler.
func NewIssueHandler(collectorURL string, cache cache.Cache, db *ent.Client) *IssueHandler {
	return &IssueHandler{
		collectorURL: collectorURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
		db:    db,
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

		// Get authenticated user from context
		user := middleware.GetUser(ctx)
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		// Validate Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType == "" || !strings.HasPrefix(contentType, "application/json") {
			writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
			return
		}

		// Limit request body size to prevent DoS attacks
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

		// Parse request body
		var req IssueSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Check if the error is due to body size limit
			if err.Error() == "http: request body too large" {
				writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body exceeds maximum size of 1MB")
				return
			}
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
		cacheKey := domain.CacheKeyIssue + issueKey

		// Use SetNX to atomically claim this issue for processing (prevents race conditions)
		claimed, err := h.cache.SetNX(ctx, cacheKey, []byte("processing"), domain.DefaultCacheTTL)
		if err != nil {
			log.Error("cache SetNX failed",
				"issue_key", issueKey,
				"error", err,
			)
			writeError(w, http.StatusInternalServerError, "cache_error", "Failed to check issue status")
			return
		}
		if !claimed {
			// Key already exists - issue is being processed or was processed
			writeError(w, http.StatusConflict, "already_processed", "This issue is already being processed")
			return
		}

		// Call collector service to fetch and process the issue (include user_id)
		collectorErr := h.callCollector(ctx, req.URL, user.ID.String())
		if collectorErr != nil {
			log.Error("collector service call failed",
				"issue_key", issueKey,
				"user_id", user.ID,
				"error", collectorErr,
			)
			// Delete the cache key so the user can retry
			if delErr := h.cache.Delete(ctx, cacheKey); delErr != nil {
				log.Warn("failed to delete cache key after collector failure",
					"issue_key", issueKey,
					"error", delErr,
				)
			}
			writeError(w, http.StatusInternalServerError, "collector_error", "Failed to process issue")
			return
		}

		// Best-effort: seed a project concept in parallel (non-blocking).
		h.startProjectConceptSeed(issueURL.FullName())

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

// startProjectConceptSeed creates a minimal project concept record if missing.
// This is best-effort and runs in a background goroutine.
func (h *IssueHandler) startProjectConceptSeed(fullName string) {
	if fullName == "" || h.db == nil {
		return
	}
	slug := slugifyProject(fullName)
	if slug == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := h.db.Concept.Query().
			Where(concept.SlugEQ(slug)).
			Only(ctx)
		if err == nil {
			return
		}
		if !ent.IsNotFound(err) {
			log.Warn("concept seed lookup failed", "slug", slug, "error", err)
			return
		}

		name := fullName
		desc := fmt.Sprintf("Open-source project %s.", fullName)
		entry := concepts.Entry{
			Slug:        slug,
			Name:        name,
			Description: desc,
			Category:    "project",
		}
		tutorial := concepts.BuildTutorialMarkdown(entry)
		if _, err := h.db.Concept.Create().
			SetSlug(slug).
			SetName(name).
			SetDescription(desc).
			SetCategory("project").
			SetTutorialMarkdown(tutorial).
			Save(ctx); err != nil && !ent.IsConstraintError(err) {
			log.Warn("concept seed create failed", "slug", slug, "error", err)
		}
	}()
}

func slugifyProject(input string) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// callCollector calls the collector service to fetch and process an issue.
func (h *IssueHandler) callCollector(ctx context.Context, issueURL string, userID string) error {
	payload := map[string]string{
		"url":     issueURL,
		"user_id": userID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.collectorURL+"/api/collect", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("collector request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil {
			return fmt.Errorf("collector error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("collector returned status %d", resp.StatusCode)
	}

	return nil
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
