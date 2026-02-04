package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-github/v60/github"
)

// mockGitHubServer creates a test server that mocks GitHub API responses.
func mockGitHubServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ClientConfig
		wantErr error
	}{
		{
			name:    "empty token returns error",
			cfg:     ClientConfig{Token: ""},
			wantErr: ErrEmptyToken,
		},
		{
			name: "valid token creates client",
			cfg: ClientConfig{
				Token:   "ghp_test_token",
				Timeout: 10 * time.Second,
			},
			wantErr: nil,
		},
		{
			name: "default timeout when not specified",
			cfg: ClientConfig{
				Token: "ghp_test_token",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("NewClient() unexpected error = %v", err)
				return
			}
			if client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		repo    string
		number  int
		wantErr error
	}{
		{
			name:    "valid params",
			owner:   "owner",
			repo:    "repo",
			number:  1,
			wantErr: nil,
		},
		{
			name:    "empty owner",
			owner:   "",
			repo:    "repo",
			number:  1,
			wantErr: ErrInvalidOwner,
		},
		{
			name:    "empty repo",
			owner:   "owner",
			repo:    "",
			number:  1,
			wantErr: ErrInvalidRepo,
		},
		{
			name:    "zero issue number",
			owner:   "owner",
			repo:    "repo",
			number:  0,
			wantErr: ErrInvalidIssueNumber,
		},
		{
			name:    "negative issue number",
			owner:   "owner",
			repo:    "repo",
			number:  -1,
			wantErr: ErrInvalidIssueNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParams(tt.owner, tt.repo, tt.number)
			if err != tt.wantErr {
				t.Errorf("validateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToDomainIssue(t *testing.T) {
	id := int64(123)
	number := 42
	title := "Test Issue"
	body := "Test body content"
	state := "open"
	htmlURL := "https://github.com/owner/repo/issues/42"
	labelName := "bug"

	ghIssue := &github.Issue{
		ID:      &id,
		Number:  &number,
		Title:   &title,
		Body:    &body,
		State:   &state,
		HTMLURL: &htmlURL,
		Labels: []*github.Label{
			{Name: &labelName},
		},
	}

	t.Run("converts github issue to domain issue", func(t *testing.T) {
		result := toDomainIssue(ghIssue, "owner", "repo")

		if result.ID != id {
			t.Errorf("ID = %v, want %v", result.ID, id)
		}
		if result.Number != number {
			t.Errorf("Number = %v, want %v", result.Number, number)
		}
		if result.Title != title {
			t.Errorf("Title = %v, want %v", result.Title, title)
		}
		if result.Body != body {
			t.Errorf("Body = %v, want %v", result.Body, body)
		}
		if result.State != state {
			t.Errorf("State = %v, want %v", result.State, state)
		}
		if result.HTMLURL != htmlURL {
			t.Errorf("HTMLURL = %v, want %v", result.HTMLURL, htmlURL)
		}
		if result.RepoOwner != "owner" {
			t.Errorf("RepoOwner = %v, want %v", result.RepoOwner, "owner")
		}
		if result.RepoName != "repo" {
			t.Errorf("RepoName = %v, want %v", result.RepoName, "repo")
		}
		if result.RepoFullName != "owner/repo" {
			t.Errorf("RepoFullName = %v, want %v", result.RepoFullName, "owner/repo")
		}
		if len(result.Labels) != 1 || result.Labels[0] != labelName {
			t.Errorf("Labels = %v, want [%v]", result.Labels, labelName)
		}
	})

	t.Run("handles nil issue", func(t *testing.T) {
		result := toDomainIssue(nil, "owner", "repo")
		if result != nil {
			t.Errorf("toDomainIssue(nil) = %v, want nil", result)
		}
	})
}

func TestToDomainRepository(t *testing.T) {
	id := int64(456)
	fullName := "owner/repo"
	ownerLogin := "owner"
	name := "repo"
	language := "Go"
	stars := 100
	description := "Test repository"

	ghRepo := &github.Repository{
		ID:              &id,
		FullName:        &fullName,
		Owner:           &github.User{Login: &ownerLogin},
		Name:            &name,
		Language:        &language,
		StargazersCount: &stars,
		Description:     &description,
	}

	t.Run("converts github repo to domain repo", func(t *testing.T) {
		result := toDomainRepository(ghRepo)

		if result.ID != id {
			t.Errorf("ID = %v, want %v", result.ID, id)
		}
		if result.FullName != fullName {
			t.Errorf("FullName = %v, want %v", result.FullName, fullName)
		}
		if result.Owner != ownerLogin {
			t.Errorf("Owner = %v, want %v", result.Owner, ownerLogin)
		}
		if result.Name != name {
			t.Errorf("Name = %v, want %v", result.Name, name)
		}
		if result.Language != language {
			t.Errorf("Language = %v, want %v", result.Language, language)
		}
		if result.Stars != stars {
			t.Errorf("Stars = %v, want %v", result.Stars, stars)
		}
		if result.Description != description {
			t.Errorf("Description = %v, want %v", result.Description, description)
		}
	})

	t.Run("handles nil repository", func(t *testing.T) {
		result := toDomainRepository(nil)
		if result != nil {
			t.Errorf("toDomainRepository(nil) = %v, want nil", result)
		}
	})
}

func TestHandleError(t *testing.T) {
	client, _ := NewClient(ClientConfig{Token: "test"})

	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{
			name:       "404 returns ErrNotFound",
			statusCode: http.StatusNotFound,
			wantErr:    ErrNotFound,
		},
		{
			name:       "401 returns ErrUnauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    ErrUnauthorized,
		},
		{
			name:       "403 returns ErrForbidden",
			statusCode: http.StatusForbidden,
			wantErr:    ErrForbidden,
		},
		{
			name:       "500 returns ErrGitHubUnavailable",
			statusCode: http.StatusInternalServerError,
			wantErr:    ErrGitHubUnavailable,
		},
		{
			name:       "502 returns ErrGitHubUnavailable",
			statusCode: http.StatusBadGateway,
			wantErr:    ErrGitHubUnavailable,
		},
		{
			name:       "503 returns ErrGitHubUnavailable",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    ErrGitHubUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &github.Response{
				Response: &http.Response{
					StatusCode: tt.statusCode,
				},
			}

			inputErr := errors.New("test error")
			err := client.handleError(inputErr, resp, "test operation", "owner", "repo", 123)

			if err == nil {
				t.Error("handleError() returned nil error")
				return
			}

			// Check that the error contains the expected sentinel error
			errStr := err.Error()
			if tt.wantErr != nil {
				wantStr := tt.wantErr.Error()
				if !contains(errStr, wantStr) {
					t.Errorf("handleError() error = %v, should contain %v", errStr, wantStr)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRateLimitObserver(t *testing.T) {
	t.Run("NewRateLimitObserver sets defaults", func(t *testing.T) {
		obs := NewRateLimitObserver()
		if obs.Limit != 5000 {
			t.Errorf("Limit = %v, want 5000", obs.Limit)
		}
		if obs.Remaining != 5000 {
			t.Errorf("Remaining = %v, want 5000", obs.Remaining)
		}
	})

	t.Run("UpdateFromResponse updates values", func(t *testing.T) {
		obs := NewRateLimitObserver()
		resetTime := time.Now().Add(time.Hour)
		resp := &github.Response{
			Rate: github.Rate{
				Limit:     5000,
				Remaining: 4999,
				Reset:     github.Timestamp{Time: resetTime},
			},
		}

		obs.UpdateFromResponse(resp)

		if obs.Remaining != 4999 {
			t.Errorf("Remaining = %v, want 4999", obs.Remaining)
		}
	})

	t.Run("ShouldWait returns true when low", func(t *testing.T) {
		obs := NewRateLimitObserver()
		obs.Remaining = 5

		if !obs.ShouldWait() {
			t.Error("ShouldWait() = false, want true when Remaining < pauseThreshold")
		}
	})

	t.Run("ShouldWait returns false when ok", func(t *testing.T) {
		obs := NewRateLimitObserver()
		obs.Remaining = 100

		if obs.ShouldWait() {
			t.Error("ShouldWait() = true, want false when Remaining >= pauseThreshold")
		}
	})

	t.Run("WaitIfNeeded respects context cancellation", func(t *testing.T) {
		obs := NewRateLimitObserver()
		obs.Remaining = 1
		obs.Reset = time.Now().Add(time.Hour) // Long wait

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := obs.WaitIfNeeded(ctx, nil)
		if err != context.Canceled {
			t.Errorf("WaitIfNeeded() error = %v, want context.Canceled", err)
		}
	})
}
