package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/issuesight/issuesight/backend/collector/github"
	"github.com/issuesight/issuesight/backend/collector/parser"
)

// mockCollector implements the Collector interface for testing.
type mockCollector struct {
	collectFunc func(ctx context.Context, url string, userID string) error
}

func (m *mockCollector) CollectIssue(ctx context.Context, url string, userID string) error {
	if m.collectFunc != nil {
		return m.collectFunc(ctx, url, userID)
	}
	return nil
}

func TestCollect(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		collectErr     error
		wantStatus     int
		wantErrorCode  string
	}{
		{
			name:       "successful collection",
			method:     http.MethodPost,
			body:       CollectRequest{URL: "https://github.com/owner/repo/issues/1"},
			wantStatus: http.StatusAccepted,
		},
		{
			name:          "method not allowed - GET",
			method:        http.MethodGet,
			body:          nil,
			wantStatus:    http.StatusMethodNotAllowed,
			wantErrorCode: "method_not_allowed",
		},
		{
			name:          "method not allowed - PUT",
			method:        http.MethodPut,
			body:          CollectRequest{URL: "https://github.com/owner/repo/issues/1"},
			wantStatus:    http.StatusMethodNotAllowed,
			wantErrorCode: "method_not_allowed",
		},
		{
			name:          "invalid JSON body",
			method:        http.MethodPost,
			body:          "not-valid-json",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_json",
		},
		{
			name:          "missing URL",
			method:        http.MethodPost,
			body:          CollectRequest{URL: ""},
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "missing_url",
		},
		{
			name:          "invalid URL - parser error",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "not-a-url"},
			collectErr:    parser.ErrInvalidURL,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_url",
		},
		{
			name:          "pull request URL",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "https://github.com/owner/repo/pull/1"},
			collectErr:    parser.ErrPullRequestURL,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "pull_request_not_supported",
		},
		{
			name:          "issue not found",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "https://github.com/owner/repo/issues/999"},
			collectErr:    github.ErrNotFound,
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "issue_not_found",
		},
		{
			name:          "github unauthorized",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "https://github.com/owner/repo/issues/1"},
			collectErr:    github.ErrUnauthorized,
			wantStatus:    http.StatusBadGateway,
			wantErrorCode: "github_auth_failed",
		},
		{
			name:          "github forbidden",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "https://github.com/owner/repo/issues/1"},
			collectErr:    github.ErrForbidden,
			wantStatus:    http.StatusForbidden,
			wantErrorCode: "github_access_denied",
		},
		{
			name:          "github rate limited",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "https://github.com/owner/repo/issues/1"},
			collectErr:    github.ErrRateLimited,
			wantStatus:    http.StatusTooManyRequests,
			wantErrorCode: "github_rate_limited",
		},
		{
			name:          "github unavailable",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "https://github.com/owner/repo/issues/1"},
			collectErr:    github.ErrGitHubUnavailable,
			wantStatus:    http.StatusBadGateway,
			wantErrorCode: "github_unavailable",
		},
		{
			name:          "internal server error",
			method:        http.MethodPost,
			body:          CollectRequest{URL: "https://github.com/owner/repo/issues/1"},
			collectErr:    errors.New("unexpected error"),
			wantStatus:    http.StatusInternalServerError,
			wantErrorCode: "collection_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock collector
			collector := &mockCollector{
				collectFunc: func(ctx context.Context, url string, userID string) error {
					return tt.collectErr
				},
			}

			handler := NewCollectHandler(collector)

			// Create request body
			var bodyReader *bytes.Reader
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					bodyReader = bytes.NewReader([]byte(v))
				default:
					bodyBytes, _ := json.Marshal(v)
					bodyReader = bytes.NewReader(bodyBytes)
				}
			} else {
				bodyReader = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tt.method, "/api/collect", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.Collect().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.wantStatus)
			}

			// Check error response if expected
			if tt.wantErrorCode != "" {
				var errResp ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
					t.Errorf("failed to unmarshal error response: %v", err)
				}
				if errResp.Error != tt.wantErrorCode {
					t.Errorf("error code = %v, want %v", errResp.Error, tt.wantErrorCode)
				}
			}

			// Check success response
			if tt.wantStatus == http.StatusAccepted {
				var resp CollectResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if resp.Status != "queued" {
					t.Errorf("status = %v, want queued", resp.Status)
				}
			}
		})
	}
}

func TestMapCollectError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "parser empty URL",
			err:        parser.ErrEmptyURL,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_url",
		},
		{
			name:       "parser invalid URL",
			err:        parser.ErrInvalidURL,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_url",
		},
		{
			name:       "parser invalid host",
			err:        parser.ErrInvalidHost,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_url",
		},
		{
			name:       "parser invalid path",
			err:        parser.ErrInvalidPath,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_url",
		},
		{
			name:       "parser invalid issue number",
			err:        parser.ErrInvalidIssueNumber,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_url",
		},
		{
			name:       "parser pull request",
			err:        parser.ErrPullRequestURL,
			wantStatus: http.StatusBadRequest,
			wantCode:   "pull_request_not_supported",
		},
		{
			name:       "github not found",
			err:        github.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "issue_not_found",
		},
		{
			name:       "github unauthorized",
			err:        github.ErrUnauthorized,
			wantStatus: http.StatusBadGateway,
			wantCode:   "github_auth_failed",
		},
		{
			name:       "github forbidden",
			err:        github.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantCode:   "github_access_denied",
		},
		{
			name:       "github rate limited",
			err:        github.ErrRateLimited,
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "github_rate_limited",
		},
		{
			name:       "github unavailable",
			err:        github.ErrGitHubUnavailable,
			wantStatus: http.StatusBadGateway,
			wantCode:   "github_unavailable",
		},
		{
			name:       "unknown error",
			err:        errors.New("something unexpected"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "collection_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code := mapCollectError(tt.err)
			if status != tt.wantStatus {
				t.Errorf("status = %v, want %v", status, tt.wantStatus)
			}
			if code != tt.wantCode {
				t.Errorf("code = %v, want %v", code, tt.wantCode)
			}
		})
	}
}
