// Package main provides the core service logic for the AI Processor.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/issuesight/issuesight/backend/ai-processor/llm"
	"github.com/issuesight/issuesight/backend/ai-processor/transformer"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorialcontent"
	"github.com/issuesight/issuesight/internal/platform/stream"
)

// Service errors.
var (
	ErrNilDependency     = errors.New("ai-processor: required dependency is nil")
	ErrProcessingFailed  = errors.New("ai-processor: message processing failed")
	ErrPersistenceFailed = errors.New("ai-processor: failed to persist tutorial")
)

// Service is the core AI processor that consumes GitHub issue events
// and generates tutorials using LLMs.
type Service struct {
	consumer  stream.Consumer
	llmClient *llm.Client
	db        *ent.Client
	logger    *slog.Logger

	// Configuration
	streamName   string
	groupName    string
	consumerName string
}

// ServiceConfig holds configuration for the AI processor service.
type ServiceConfig struct {
	Consumer     stream.Consumer
	LLMClient    *llm.Client
	DB           *ent.Client
	Logger       *slog.Logger
	StreamName   string // Redis stream name (default: "github-events")
	GroupName    string // Consumer group name (default: "ai-processor-group")
	ConsumerName string // This consumer's name (default: hostname-based)
}

// NewService creates a new AI processor service.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Consumer == nil {
		return nil, fmt.Errorf("%w: consumer", ErrNilDependency)
	}
	if cfg.LLMClient == nil {
		return nil, fmt.Errorf("%w: llm client", ErrNilDependency)
	}
	if cfg.DB == nil {
		return nil, fmt.Errorf("%w: database", ErrNilDependency)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.StreamName == "" {
		cfg.StreamName = domain.StreamGitHubEvents
	}
	if cfg.GroupName == "" {
		cfg.GroupName = domain.GroupAIProcessor
	}
	if cfg.ConsumerName == "" {
		cfg.ConsumerName = fmt.Sprintf("ai-processor-%s", uuid.New().String()[:8])
	}

	return &Service{
		consumer:     cfg.Consumer,
		llmClient:    cfg.LLMClient,
		db:           cfg.DB,
		logger:       cfg.Logger,
		streamName:   cfg.StreamName,
		groupName:    cfg.GroupName,
		consumerName: cfg.ConsumerName,
	}, nil
}

// Start begins consuming messages from the stream.
// This method blocks until the context is cancelled.
func (s *Service) Start(ctx context.Context) error {
	// Create consumer group if it doesn't exist
	if err := s.consumer.CreateGroup(ctx, s.streamName, s.groupName); err != nil {
		return fmt.Errorf("create consumer group: %w", err)
	}

	s.logger.Info("starting stream consumer",
		"stream", s.streamName,
		"group", s.groupName,
		"consumer", s.consumerName,
	)

	// Start consuming - this blocks until context is cancelled
	return s.consumer.Consume(ctx, s.streamName, s.groupName, s.consumerName, s.handleMessage)
}

// handleMessage processes a single message from the stream.
func (s *Service) handleMessage(msg stream.Message) error {
	start := time.Now()
	ctx := context.Background()

	s.logger.Debug("processing message",
		"id", msg.ID,
		"stream", msg.Stream,
	)

	// 1. Transform message to stream payload (wire format)
	payload, err := transformer.TransformStreamMessage(msg.Payload)
	if err != nil {
		s.logger.Error("failed to transform message",
			"id", msg.ID,
			"error", err,
		)
		// Return nil to acknowledge - malformed messages shouldn't be retried
		return nil
	}

	s.logger.Info("processing issue",
		"issue_id", payload.IssueID,
		"repo", payload.FullName,
		"title", truncateLog(payload.Title, 50),
	)

	// 2. Check idempotency - skip if already completed
	existing, err := s.db.TutorialContent.Query().
		Where(tutorialcontent.IssueIDEQ(uuid.MustParse(fmt.Sprintf("%036d", payload.IssueID)))).
		Only(ctx)
	if err == nil && existing.Status == tutorialcontent.StatusCompleted {
		s.logger.Info("tutorial already exists, skipping",
			"issue_id", payload.IssueID,
			"tutorial_id", existing.ID,
		)
		return nil
	}

	// 3. Generate tutorial content using LLM
	content, err := s.generateTutorial(ctx, payload)
	if err != nil {
		s.logger.Error("failed to generate tutorial",
			"issue_id", payload.IssueID,
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		// Mark as failed if we have a record
		if existing != nil {
			s.markAsFailed(ctx, existing.ID)
		}
		return fmt.Errorf("%w: %v", ErrProcessingFailed, err)
	}

	// 4. Parse LLM output to domain.TutorialContent
	tutorial, err := transformer.ParseLLMOutput(content)
	if err != nil {
		s.logger.Error("failed to parse LLM output",
			"issue_id", payload.IssueID,
			"error", err,
		)
		if existing != nil {
			s.markAsFailed(ctx, existing.ID)
		}
		return nil // Don't retry parsing failures
	}

	// 5. Persist to database
	if err := s.persistTutorial(ctx, payload.IssueID, tutorial); err != nil {
		s.logger.Error("failed to persist tutorial",
			"issue_id", payload.IssueID,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrPersistenceFailed, err)
	}

	s.logger.Info("tutorial generated successfully",
		"issue_id", payload.IssueID,
		"title", truncateLog(tutorial.Title, 50),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

// generateTutorial calls the LLM to generate tutorial content.
func (s *Service) generateTutorial(ctx context.Context, payload *transformer.StreamIssuePayload) (string, error) {
	userPrompt := llm.BuildUserPrompt(payload)

	return s.llmClient.GenerateWithSystem(ctx, llm.SystemPrompt, userPrompt)
}

// persistTutorial saves the tutorial content to the database.
func (s *Service) persistTutorial(ctx context.Context, issueID int64, tutorial *domain.TutorialContent) error {
	// Create tutorial content
	_, err := s.db.TutorialContent.Create().
		SetIssueID(uuid.MustParse(fmt.Sprintf("%036d", issueID))). // Convert issue ID to UUID format
		SetTitle(tutorial.Title).
		SetMarkdownBody(tutorial.MarkdownBody).
		SetStatus(tutorialcontent.StatusCompleted).
		Save(ctx)

	if err != nil {
		// Check if it's a unique constraint violation (already exists)
		if ent.IsConstraintError(err) {
			// Update existing record instead
			return s.updateExistingTutorial(ctx, issueID, tutorial)
		}
		return err
	}

	return nil
}

// updateExistingTutorial updates an existing tutorial content record.
func (s *Service) updateExistingTutorial(ctx context.Context, issueID int64, tutorial *domain.TutorialContent) error {
	_, err := s.db.TutorialContent.Update().
		Where(tutorialcontent.IssueIDEQ(uuid.MustParse(fmt.Sprintf("%036d", issueID)))).
		SetTitle(tutorial.Title).
		SetMarkdownBody(tutorial.MarkdownBody).
		SetStatus(tutorialcontent.StatusCompleted).
		Save(ctx)
	return err
}

// markAsFailed updates the tutorial status to failed.
func (s *Service) markAsFailed(ctx context.Context, id uuid.UUID) {
	_, err := s.db.TutorialContent.UpdateOneID(id).
		SetStatus(tutorialcontent.StatusFailed).
		Save(ctx)
	if err != nil {
		s.logger.Error("failed to mark tutorial as failed",
			"id", id,
			"error", err,
		)
	}
}

// truncateLog truncates a string for logging purposes.
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
