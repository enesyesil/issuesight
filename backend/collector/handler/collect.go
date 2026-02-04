package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/issuesight/issuesight/backend/collector/github"
	"github.com/issuesight/issuesight/backend/collector/parser"
)

// CollectRequest is the request body for the collect endpoint.
type CollectRequest struct {
	URL    string `json:"url"`
	UserID string `json:"user_id"`
}

// CollectResponse is the response body for the collect endpoint.
type CollectResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ErrorResponse is the response body for errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Collector interface for the service.
type Collector interface {
	CollectIssue(ctx context.Context, url string, userID string) error
}

// CollectHandler handles issue collection requests.
type CollectHandler struct {
	service Collector
}

// NewCollectHandler creates a new collect handler.
func NewCollectHandler(service Collector) *CollectHandler {
	return &CollectHandler{service: service}
}

// Collect returns an HTTP handler for the collect endpoint.
func (h *CollectHandler) Collect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
			return
		}

		var req CollectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
			return
		}

		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "missing_url", "URL is required")
			return
		}

		if err := h.service.CollectIssue(r.Context(), req.URL, req.UserID); err != nil {
			status, code := mapCollectError(err)
			writeError(w, status, code, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(CollectResponse{
			Status:  "queued",
			Message: "Issue queued for processing",
		})
	}
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// mapCollectError maps service errors to HTTP status codes and error codes.
func mapCollectError(err error) (int, string) {
	// Parser errors - bad request
	if errors.Is(err, parser.ErrEmptyURL) ||
		errors.Is(err, parser.ErrInvalidURL) ||
		errors.Is(err, parser.ErrInvalidHost) ||
		errors.Is(err, parser.ErrInvalidPath) ||
		errors.Is(err, parser.ErrInvalidIssueNumber) {
		return http.StatusBadRequest, "invalid_url"
	}
	if errors.Is(err, parser.ErrPullRequestURL) {
		return http.StatusBadRequest, "pull_request_not_supported"
	}

	// GitHub errors
	if errors.Is(err, github.ErrNotFound) {
		return http.StatusNotFound, "issue_not_found"
	}
	if errors.Is(err, github.ErrUnauthorized) {
		return http.StatusBadGateway, "github_auth_failed"
	}
	if errors.Is(err, github.ErrForbidden) {
		return http.StatusForbidden, "github_access_denied"
	}
	if errors.Is(err, github.ErrRateLimited) {
		return http.StatusTooManyRequests, "github_rate_limited"
	}
	if errors.Is(err, github.ErrGitHubUnavailable) {
		return http.StatusBadGateway, "github_unavailable"
	}

	// Default to internal server error
	return http.StatusInternalServerError, "collection_failed"
}
