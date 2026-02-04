// Package http provides shared HTTP utilities for all services.
package http

import (
	"encoding/json"
	"net/http"

	"github.com/issuesight/issuesight/internal/platform/log"
)

// ErrorResponse represents a standard JSON error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteJSON writes a JSON response with the given status code.
// It sets the Content-Type header and encodes the data as JSON.
// Returns an error if encoding fails (but response is still sent).
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error("failed to encode JSON response",
			"status", status,
			"error", err,
		)
		return err
	}
	return nil
}

// WriteError writes a standard JSON error response.
// This provides a consistent error format across all services.
func WriteError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}
