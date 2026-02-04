package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/lock"
)

// mockGitHubClient implements a mock GitHub client for testing.
type mockGitHubClient struct {
	fetchIssueFunc      func(ctx context.Context, owner, repo string, number int) (*domain.Issue, error)
	fetchRepositoryFunc func(ctx context.Context, owner, repo string) (*domain.Repository, error)
}

func (m *mockGitHubClient) FetchIssue(ctx context.Context, owner, repo string, number int) (*domain.Issue, error) {
	if m.fetchIssueFunc != nil {
		return m.fetchIssueFunc(ctx, owner, repo, number)
	}
	return &domain.Issue{
		ID:           123,
		Number:       number,
		Title:        "Test Issue",
		Body:         "Test body",
		RepoOwner:    owner,
		RepoName:     repo,
		RepoFullName: owner + "/" + repo,
		Labels:       []string{"bug"},
		State:        "open",
		HTMLURL:      fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, number),
	}, nil
}

func (m *mockGitHubClient) FetchRepository(ctx context.Context, owner, repo string) (*domain.Repository, error) {
	if m.fetchRepositoryFunc != nil {
		return m.fetchRepositoryFunc(ctx, owner, repo)
	}
	return &domain.Repository{
		ID:       456,
		FullName: owner + "/" + repo,
		Owner:    owner,
		Name:     repo,
		Language: "Go",
		Stars:    100,
	}, nil
}

// mockPublisher implements a mock stream publisher for testing.
type mockPublisher struct {
	publishFunc func(ctx context.Context, stream string, payload map[string]interface{}) (string, error)
	published   []map[string]interface{}
}

func (m *mockPublisher) Publish(ctx context.Context, stream string, payload map[string]interface{}) (string, error) {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, stream, payload)
	}
	m.published = append(m.published, payload)
	return "msg-123", nil
}

// mockLocker implements a mock distributed lock for testing.
type mockLocker struct {
	acquireFunc func(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error)
	lockObj     *mockLock
}

func (m *mockLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error) {
	if m.acquireFunc != nil {
		return m.acquireFunc(ctx, key, ttl)
	}
	if m.lockObj == nil {
		m.lockObj = &mockLock{key: key}
	}
	return m.lockObj, nil
}

func (m *mockLocker) TryAcquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error) {
	return m.Acquire(ctx, key, ttl)
}

func (m *mockLocker) AcquireWithRetry(ctx context.Context, key string, ttl time.Duration, retries int, retryDelay time.Duration) (lock.Lock, error) {
	return m.Acquire(ctx, key, ttl)
}

type mockLock struct {
	releaseFunc func(ctx context.Context) error
	released    bool
	key         string
}

func (h *mockLock) Release(ctx context.Context) error {
	if h.releaseFunc != nil {
		return h.releaseFunc(ctx)
	}
	h.released = true
	return nil
}

func (h *mockLock) Extend(ctx context.Context, ttl time.Duration) error {
	return nil
}

func (h *mockLock) Key() string {
	return h.key
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServiceConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config creates service",
			cfg: ServiceConfig{
				GitHub:    &mockGitHubClient{},
				Publisher: &mockPublisher{},
				Locker:    &mockLocker{},
				Logger:    slog.Default(),
			},
			wantErr: false,
		},
		{
			name: "missing github returns error",
			cfg: ServiceConfig{
				Publisher: &mockPublisher{},
				Locker:    &mockLocker{},
			},
			wantErr: true,
			errMsg:  "github client is required",
		},
		{
			name: "missing publisher returns error",
			cfg: ServiceConfig{
				GitHub: &mockGitHubClient{},
				Locker: &mockLocker{},
			},
			wantErr: true,
			errMsg:  "publisher is required",
		},
		{
			name: "missing locker returns error",
			cfg: ServiceConfig{
				GitHub:    &mockGitHubClient{},
				Publisher: &mockPublisher{},
			},
			wantErr: true,
			errMsg:  "locker is required",
		},
		{
			name: "nil logger uses default",
			cfg: ServiceConfig{
				GitHub:    &mockGitHubClient{},
				Publisher: &mockPublisher{},
				Locker:    &mockLocker{},
				Logger:    nil,
			},
			wantErr: false,
		},
		{
			name: "default stream name and lock TTL",
			cfg: ServiceConfig{
				GitHub:    &mockGitHubClient{},
				Publisher: &mockPublisher{},
				Locker:    &mockLocker{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("NewService() expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("NewService() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("NewService() unexpected error = %v", err)
				return
			}
			if svc == nil {
				t.Error("NewService() returned nil service")
			}
		})
	}
}

func TestCollectIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("successful collection", func(t *testing.T) {
		publisher := &mockPublisher{}
		locker := &mockLocker{}

		svc, err := NewService(ServiceConfig{
			GitHub:    &mockGitHubClient{},
			Publisher: publisher,
			Locker:    locker,
			Logger:    slog.Default(),
		})
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		err = svc.CollectIssue(ctx, "https://github.com/owner/repo/issues/42", "test-user-id")
		if err != nil {
			t.Errorf("CollectIssue() error = %v", err)
		}

		if len(publisher.published) != 1 {
			t.Errorf("expected 1 published message, got %d", len(publisher.published))
		}

		if locker.lockObj != nil && !locker.lockObj.released {
			t.Error("lock was not released")
		}
	})

	t.Run("invalid URL returns error", func(t *testing.T) {
		svc, _ := NewService(ServiceConfig{
			GitHub:    &mockGitHubClient{},
			Publisher: &mockPublisher{},
			Locker:    &mockLocker{},
			Logger:    slog.Default(),
		})

		err := svc.CollectIssue(ctx, "not-a-valid-url", "test-user-id")
		if err == nil {
			t.Error("CollectIssue() expected error for invalid URL")
		}
	})

	t.Run("lock already held returns duplicate error", func(t *testing.T) {
		locker := &mockLocker{
			acquireFunc: func(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error) {
				return nil, lock.ErrLockNotAcquired
			},
		}

		svc, _ := NewService(ServiceConfig{
			GitHub:    &mockGitHubClient{},
			Publisher: &mockPublisher{},
			Locker:    locker,
			Logger:    slog.Default(),
		})

		err := svc.CollectIssue(ctx, "https://github.com/owner/repo/issues/42", "test-user-id")
		if !errors.Is(err, ErrDuplicateRequest) {
			t.Errorf("CollectIssue() error = %v, want %v", err, ErrDuplicateRequest)
		}
	})

	t.Run("github fetch failure", func(t *testing.T) {
		ghClient := &mockGitHubClient{
			fetchRepositoryFunc: func(ctx context.Context, owner, repo string) (*domain.Repository, error) {
				return nil, errors.New("github unavailable")
			},
		}

		svc, _ := NewService(ServiceConfig{
			GitHub:    ghClient,
			Publisher: &mockPublisher{},
			Locker:    &mockLocker{},
			Logger:    slog.Default(),
		})

		err := svc.CollectIssue(ctx, "https://github.com/owner/repo/issues/42", "test-user-id")
		if err == nil {
			t.Error("CollectIssue() expected error for GitHub failure")
		}
	})

	t.Run("publish failure", func(t *testing.T) {
		publisher := &mockPublisher{
			publishFunc: func(ctx context.Context, stream string, payload map[string]interface{}) (string, error) {
				return "", errors.New("publish failed")
			},
		}

		svc, _ := NewService(ServiceConfig{
			GitHub:    &mockGitHubClient{},
			Publisher: publisher,
			Locker:    &mockLocker{},
			Logger:    slog.Default(),
		})

		err := svc.CollectIssue(ctx, "https://github.com/owner/repo/issues/42", "test-user-id")
		if err == nil {
			t.Error("CollectIssue() expected error for publish failure")
		}
		if !errors.Is(err, ErrPublishFailed) {
			t.Errorf("CollectIssue() error = %v, want %v", err, ErrPublishFailed)
		}
	})
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "string equal to max",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "string longer than max",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "maxLen less than 4",
			input:  "hello",
			maxLen: 3,
			want:   "",
		},
		{
			name:   "maxLen equals 4",
			input:  "hello",
			maxLen: 4,
			want:   "h...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestBuildPayload(t *testing.T) {
	svc, _ := NewService(ServiceConfig{
		GitHub:    &mockGitHubClient{},
		Publisher: &mockPublisher{},
		Locker:    &mockLocker{},
		Logger:    slog.Default(),
	})

	issue := &domain.Issue{
		ID:           123,
		Number:       42,
		Title:        "Test Issue",
		Body:         "Test body",
		RepoOwner:    "owner",
		RepoName:     "repo",
		RepoFullName: "owner/repo",
		Labels:       []string{"bug", "enhancement"},
		State:        "open",
		HTMLURL:      "https://github.com/owner/repo/issues/42",
	}

	repo := &domain.Repository{
		ID:       456,
		Language: "Go",
		Stars:    100,
	}

	payload := svc.buildPayload(issue, repo, "test-user-id")

	// Check required fields
	if payload["issue_id"] != int64(123) {
		t.Errorf("issue_id = %v, want 123", payload["issue_id"])
	}
	if payload["issue_number"] != 42 {
		t.Errorf("issue_number = %v, want 42", payload["issue_number"])
	}
	if payload["owner"] != "owner" {
		t.Errorf("owner = %v, want owner", payload["owner"])
	}
	if payload["repo"] != "repo" {
		t.Errorf("repo = %v, want repo", payload["repo"])
	}
	if payload["title"] != "Test Issue" {
		t.Errorf("title = %v, want Test Issue", payload["title"])
	}
	if payload["repo_language"] != "Go" {
		t.Errorf("repo_language = %v, want Go", payload["repo_language"])
	}
	if payload["repo_stars"] != 100 {
		t.Errorf("repo_stars = %v, want 100", payload["repo_stars"])
	}
	if payload["user_id"] != "test-user-id" {
		t.Errorf("user_id = %v, want test-user-id", payload["user_id"])
	}

	// Check labels are JSON encoded
	labelsJSON := payload["labels"].(string)
	if labelsJSON != `["bug","enhancement"]` {
		t.Errorf("labels = %v, want [\"bug\",\"enhancement\"]", labelsJSON)
	}

	// Check collected_at is set
	if _, ok := payload["collected_at"]; !ok {
		t.Error("collected_at not set in payload")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
