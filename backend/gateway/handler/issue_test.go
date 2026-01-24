package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/issuesight/issuesight/backend/gateway/handler"
)

func TestIssueHandler_Submit_InvalidJSON(t *testing.T) {
	// Create handler with nil dependencies (we'll hit JSON parsing first)
	h := handler.NewIssueHandler(nil, nil, nil, nil)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString("not json"))
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
	h := handler.NewIssueHandler(nil, nil, nil, nil)

	body := `{"url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString(body))
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
	h := handler.NewIssueHandler(nil, nil, nil, nil)

	body := `{"url": "https://example.com/not-github"}`
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString(body))
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
	h := handler.NewIssueHandler(nil, nil, nil, nil)

	body := `{"url": "https://github.com/owner/repo/pull/123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/issues", bytes.NewBufferString(body))
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
