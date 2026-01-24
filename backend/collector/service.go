// Package collector provides the core service logic for collecting GitHub issues.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/issuesight/issuesight/backend/collector/github"
	"github.com/issuesight/issuesight/backend/collector/parser"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/lock"
	"github.com/issuesight/issuesight/internal/platform/stream"
)

// Collector service errors.
var (
	ErrDuplicateRequest = errors.New("collector: issue already being processed")
	ErrPublishFailed    = errors.New("collector: failed to publish to stream")
)

// Service is the core collector service that fetches GitHub issues
// and publishes them to Redis Streams for AI processing.
type Service struct {
	github    *github.Client
	publisher stream.Publisher
	locker    lock.Locker
	logger    *slog.Logger

	// Configuration
	streamName string
	lockTTL    time.Duration
}

// ServiceConfig holds configuration for the collector service.
type ServiceConfig struct {
	GitHub     *github.Client
	Publisher  stream.Publisher
	Locker     lock.Locker
	Logger     *slog.Logger
	StreamName string        // Redis stream name (default: "github-events")
	LockTTL    time.Duration // Lock TTL (default: 5 minutes)
}

// NewService creates a new collector service.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.GitHub == nil {
		return nil, errors.New("collector: github client is required")
	}
	if cfg.Publisher == nil {
		return nil, errors.New("collector: publisher is required")
	}
	if cfg.Locker == nil {
		return nil, errors.New("collector: locker is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.StreamName == "" {
		cfg.StreamName = domain.StreamGitHubEvents
	}
	if cfg.LockTTL == 0 {
		cfg.LockTTL = domain.DefaultLockTTL
	}

	return &Service{
		github:     cfg.GitHub,
		publisher:  cfg.Publisher,
		locker:     cfg.Locker,
		logger:     cfg.Logger,
		streamName: cfg.StreamName,
		lockTTL:    cfg.LockTTL,
	}, nil
}

// CollectIssue fetches a GitHub issue and publishes it to the stream.
//
// Flow:
//  1. Parse and validate the issue URL
//  2. Acquire a distributed lock to prevent duplicate processing
//  3. Fetch repository metadata from GitHub
//  4. Fetch issue details from GitHub
//  5. Publish to Redis Stream
//  6. Release lock
func (s *Service) CollectIssue(ctx context.Context, issueURL string) error {
	start := time.Now()

	// 1. Parse URL
	parsed, err := parser.Parse(issueURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	lockKey := fmt.Sprintf("%s%s/%s/%d", domain.LockKeyIssue, parsed.Owner, parsed.Repo, parsed.Number)

	// 2. Acquire lock
	lockHandle, err := s.locker.Acquire(ctx, lockKey, s.lockTTL)
	if err != nil {
		if errors.Is(err, lock.ErrLockNotAcquired) {
			s.logger.Warn("issue already being processed",
				"owner", parsed.Owner,
				"repo", parsed.Repo,
				"number", parsed.Number,
			)
			return ErrDuplicateRequest
		}
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() {
		if err := lockHandle.Release(ctx); err != nil {
			s.logger.Error("failed to release lock",
				"key", lockKey,
				"error", err,
			)
		}
	}()

	// 3. Fetch repository
	repo, err := s.github.FetchRepository(ctx, parsed.Owner, parsed.Repo)
	if err != nil {
		return fmt.Errorf("fetch repository: %w", err)
	}

	// 4. Fetch issue
	issue, err := s.github.FetchIssue(ctx, parsed.Owner, parsed.Repo, parsed.Number)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}

	// 5. Build payload and publish
	payload := s.buildPayload(issue, repo)

	msgID, err := s.publisher.Publish(ctx, s.streamName, payload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	s.logger.Info("issue collected",
		"owner", parsed.Owner,
		"repo", parsed.Repo,
		"number", parsed.Number,
		"message_id", msgID,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

// buildPayload creates the stream message payload.
func (s *Service) buildPayload(issue *domain.Issue, repo *domain.Repository) map[string]interface{} {
	// Convert labels to JSON string for Redis
	labelsJSON, _ := json.Marshal(issue.Labels)

	return map[string]interface{}{
		"issue_id":      issue.ID,
		"issue_number":  issue.Number,
		"owner":         issue.RepoOwner,
		"repo":          issue.RepoName,
		"full_name":     issue.RepoFullName,
		"title":         truncateString(issue.Title, 500),
		"body":          truncateString(issue.Body, 100000), // ~100KB limit
		"labels":        string(labelsJSON),
		"state":         issue.State,
		"html_url":      issue.HTMLURL,
		"repo_id":       repo.ID,
		"repo_language": repo.Language,
		"repo_stars":    repo.Stars,
		"collected_at":  time.Now().UTC().Format(time.RFC3339),
	}
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
