package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth_AllHealthy(t *testing.T) {
	// Create checker with LLM ready but no real clients
	// (nil clients are skipped in checks)
	checker := NewHealthChecker(nil, nil, true)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	checker.Health()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Status = %s, want 'healthy'", response.Status)
	}

	// All checks should be OK when clients are nil (skipped)
	for key, val := range response.Checks {
		if val != "ok" {
			t.Errorf("Check %s = %s, want 'ok'", key, val)
		}
	}
}

func TestHealth_LLMNotReady(t *testing.T) {
	checker := NewHealthChecker(nil, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	checker.Health()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("Status = %s, want 'unhealthy'", response.Status)
	}

	if response.Checks["llm"] != "not initialized" {
		t.Errorf("LLM check = %s, want 'not initialized'", response.Checks["llm"])
	}
}

func TestHealth_ContentType(t *testing.T) {
	checker := NewHealthChecker(nil, nil, true)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	checker.Health()(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want 'application/json'", contentType)
	}
}

func TestNewHealthChecker(t *testing.T) {
	checker := NewHealthChecker(nil, nil, true)
	if checker == nil {
		t.Error("NewHealthChecker returned nil")
	}
	if !checker.llmReady {
		t.Error("llmReady should be true")
	}
}
