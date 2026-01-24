package handler

import (
	"context"
	"encoding/json"
	"net/http"
)

// CollectRequest is the request body for the collect endpoint.
type CollectRequest struct {
	URL string `json:"url"`
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
	CollectIssue(ctx context.Context, url string) error
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

		if err := h.service.CollectIssue(r.Context(), req.URL); err != nil {
			// TODO: Map specific errors to appropriate status codes
			writeError(w, http.StatusInternalServerError, "collection_failed", err.Error())
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
