package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/issuesight/issuesight/backend/gateway/handler"
)

func TestHealthHandler_Healthy(t *testing.T) {
	// Create handler with nil clients (will skip checks)
	h := handler.NewHealthHandler(nil, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Call handler
	h.Health()(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Parse response body
	var resp handler.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", resp.Status)
	}

	// With nil clients, checks should be skipped (return ok)
	if redis, ok := resp.Checks["redis"]; !ok || redis != "ok" {
		t.Errorf("expected redis check 'ok', got %q", redis)
	}

	if postgres, ok := resp.Checks["postgres"]; !ok || postgres != "ok" {
		t.Errorf("expected postgres check 'ok', got %q", postgres)
	}
}

func TestHealthHandler_ContentType(t *testing.T) {
	h := handler.NewHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health()(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", contentType)
	}
}
