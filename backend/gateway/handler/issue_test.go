package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/issuesight/issuesight/backend/gateway/handler"
	"github.com/issuesight/issuesight/backend/gateway/middleware"
)

// addAuthContext adds a mock authenticated user to the request context.
func addAuthContext(req *http.Request) *http.Request {
	user := &middleware.AuthUser{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	ctx := context.WithValue(req.Context(), middleware.UserKey{}, user)
	return req.WithContext(ctx)
}

func TestIssueHandler_Submit_InvalidJSON(t *testing.T) {
	// Create handler with empty collector URL (we'll hit JSON parsing first)
	h := handler.NewIssueHandler("", nil, nil)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req = addAuthContext(req)
	rec := httptest.NewRecorder()

	h.Submit()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp handler.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "invalid_request" {
		t.Errorf("expected error 'invalid_request', got %q", resp.Error)
	}
}

func TestIssueHandler_Submit_EmptyURL(t *testing.T) {
	h := handler.NewIssueHandler("", nil, nil)

	body := `{"url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addAuthContext(req)
	rec := httptest.NewRecorder()

	h.Submit()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp handler.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "invalid_url" {
		t.Errorf("expected error 'invalid_url', got %q", resp.Error)
	}
}

func TestIssueHandler_Submit_InvalidURL(t *testing.T) {
	h := handler.NewIssueHandler("", nil, nil)

	body := `{"url": "https://example.com/not-github"}`
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addAuthContext(req)
	rec := httptest.NewRecorder()

	h.Submit()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp handler.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "invalid_url" {
		t.Errorf("expected error 'invalid_url', got %q", resp.Error)
	}

	if resp.Message != "URL must be from github.com" {
		t.Errorf("expected message about github.com, got %q", resp.Message)
	}
}

func TestIssueHandler_Submit_PullRequestURL(t *testing.T) {
	h := handler.NewIssueHandler("", nil, nil)

	body := `{"url": "https://github.com/owner/repo/pull/123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addAuthContext(req)
	rec := httptest.NewRecorder()

	h.Submit()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp handler.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "Pull request URLs are not supported" {
		t.Errorf("expected pull request error message, got %q", resp.Message)
	}
}

func TestIssueHandler_Submit_NoAuth(t *testing.T) {
	h := handler.NewIssueHandler("", nil, nil)

	body := `{"url": "https://github.com/owner/repo/issues/123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	// Note: NOT adding auth context
	rec := httptest.NewRecorder()

	h.Submit()(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
