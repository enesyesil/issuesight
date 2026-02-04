// Package main provides the core service logic for the AI Processor.
package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/issuesight/issuesight/backend/ai-processor/llm"
	"github.com/issuesight/issuesight/backend/ai-processor/transformer"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/githubissue"
	"github.com/issuesight/issuesight/internal/platform/db/ent/project"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorialcontent"
	"github.com/issuesight/issuesight/internal/platform/stream"
)

// IssueSightNamespace is the UUID v5 namespace for generating deterministic UUIDs from issue IDs.
// This ensures the same issue ID always produces the same UUID.
var IssueSightNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

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
	// Bound total processing time per message to avoid hangs.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

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

	// Ensure GithubIssue exists first (TutorialContent.issue_id is a required FK).
	if _, err := s.ensureProjectAndIssue(ctx, payload); err != nil {
		s.logger.Error("failed to ensure project/issue",
			"issue_id", payload.IssueID,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrPersistenceFailed, err)
	}

	issueUUID := issueIDToUUID(payload.IssueID)

	// Best-effort: sync project concepts in parallel (non-blocking).
	s.startProjectConceptSync(ctx, payload)

	// 2. Check idempotency - skip if already completed.
	existing, err := s.db.TutorialContent.Query().
		Where(tutorialcontent.IssueIDEQ(issueUUID)).
		Only(ctx)
	if err == nil && existing.Status == tutorialcontent.StatusCompleted {
		s.logger.Info("tutorial already exists, skipping",
			"issue_id", payload.IssueID,
			"tutorial_id", existing.ID,
		)
		return nil
	}

	// Ensure we have a row to update even if generation/validation fails.
	// This avoids the "nothing saved to Postgres" situation for failed runs.
	if existing == nil {
		title := tutorialTitlePlaceholder(payload)
		existing, err = s.db.TutorialContent.Create().
			SetIssueID(issueUUID).
			SetTitle(title).
			SetMarkdownBody("").
			SetStatus(tutorialcontent.StatusPending).
			Save(ctx)
		if err != nil {
			// Handle races where another consumer created it.
			if ent.IsConstraintError(err) {
				existing, err = s.db.TutorialContent.Query().
					Where(tutorialcontent.IssueIDEQ(issueUUID)).
					Only(ctx)
			}
			if err != nil {
				s.logger.Error("failed to create pending tutorial record",
					"issue_id", payload.IssueID,
					"error", err,
				)
				return fmt.Errorf("%w: %v", ErrPersistenceFailed, err)
			}
		}
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

	// 4. Validate LLM response before parsing
	if err := validateLLMResponse(content); err != nil {
		s.logger.Error("LLM response validation failed",
			"issue_id", payload.IssueID,
			"error", err,
			"content_length", len(content),
		)
		if existing != nil {
			s.markAsFailed(ctx, existing.ID)
		}
		return nil // Don't retry validation failures, but keep FAILED record.
	}

	// 5. Parse LLM output to domain.TutorialContent (and extract prerequisites)
	tutorial, prerequisites, err := transformer.ParseLLMOutputWithPrereqs(content)
	if err != nil {
		s.logger.Error("failed to parse LLM output",
			"issue_id", payload.IssueID,
			"error", err,
		)
		if existing != nil {
			s.markAsFailed(ctx, existing.ID)
		}
		return nil // Don't retry parsing failures, but keep FAILED record.
	}

	// 5b. Auto-fix tutorial markdown to enforce required structure.
	tutorial.MarkdownBody = transformer.AutoFixTutorialMarkdown(tutorial.MarkdownBody)
	if err := transformer.ValidateTutorialMarkdown(tutorial.MarkdownBody); err != nil {
		s.logger.Warn("tutorial markdown validation failed after auto-fix",
			"issue_id", payload.IssueID,
			"error", err,
		)
	}

	// 6. Persist to database
	if err := s.persistTutorial(ctx, payload, tutorial); err != nil {
		s.logger.Error("failed to persist tutorial",
			"issue_id", payload.IssueID,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrPersistenceFailed, err)
	}

	// 7. Generate concepts from prerequisites (best-effort)
	if err := s.syncConcepts(ctx, payload, issueUUID, prerequisites); err != nil {
		s.logger.Warn("failed to sync concepts",
			"issue_id", payload.IssueID,
			"error", err,
		)
	}

	s.logger.Info("tutorial generated successfully",
		"issue_id", payload.IssueID,
		"title", truncateLog(tutorial.Title, 50),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

func tutorialTitlePlaceholder(payload *transformer.StreamIssuePayload) string {
	// Title is NotEmpty in the schema; use a stable, human-friendly placeholder.
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		return fmt.Sprintf("Tutorial for %s#%d", payload.FullName, payload.IssueNumber)
	}
	// Keep it reasonably short for UI; DB schema doesn't enforce max length,
	// but long titles aren't useful.
	if len(title) > 200 {
		return title[:197] + "..."
	}
	return title
}

// LLM call timeout for tutorial generation.
const llmCallTimeout = 2 * time.Minute

// Project concept sync timeout (best-effort background task).
const projectConceptSyncTimeout = 30 * time.Second

// generateTutorial calls the LLM to generate tutorial content.
func (s *Service) generateTutorial(ctx context.Context, payload *transformer.StreamIssuePayload) (string, error) {
	// Add timeout for LLM calls to prevent hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, llmCallTimeout)
	defer cancel()

	userPrompt := llm.BuildUserPrompt(payload)

	return s.llmClient.GenerateWithSystem(timeoutCtx, llm.SystemPrompt, userPrompt)
}

// startProjectConceptSync kicks off a best-effort background sync for project concepts.
func (s *Service) startProjectConceptSync(parentCtx context.Context, payload *transformer.StreamIssuePayload) {
	if payload == nil {
		return
	}
	syncCtx, cancel := context.WithTimeout(parentCtx, projectConceptSyncTimeout)
	go func() {
		defer cancel()
		if err := s.syncProjectConcepts(syncCtx, payload); err != nil {
			s.logger.Warn("project concept sync failed",
				"repo", payload.FullName,
				"error", err,
			)
		}
	}()
}

// persistTutorial saves the tutorial content to the database.
func (s *Service) persistTutorial(ctx context.Context, payload *transformer.StreamIssuePayload, tutorial *domain.TutorialContent) error {
	issueUUID, err := s.ensureProjectAndIssue(ctx, payload)
	if err != nil {
		return err
	}

	// Create tutorial content (FKs require GithubIssue to exist)
	content, err := s.db.TutorialContent.Create().
		SetIssueID(issueUUID).
		SetTitle(tutorial.Title).
		SetMarkdownBody(tutorial.MarkdownBody).
		SetStatus(tutorialcontent.StatusCompleted).
		Save(ctx)

	if err != nil {
		// Only treat UNIQUE violations as "already exists". Other constraint errors
		// (e.g., foreign key violations) should be returned.
		if ent.IsConstraintError(err) && strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return s.updateExistingTutorial(ctx, issueUUID, tutorial, payload.UserID)
		}
		return err
	}

	// Create Tutorial access row to link user to content (if user_id provided)
	if payload.UserID != "" {
		if err := s.createTutorialAccess(ctx, payload.UserID, content.ID, true); err != nil {
			s.logger.Warn("failed to create tutorial access row",
				"user_id", payload.UserID,
				"content_id", content.ID,
				"error", err,
			)
			// Don't fail the whole operation - content was created
		}
	}

	return nil
}

// updateExistingTutorial updates an existing tutorial content record.
func (s *Service) updateExistingTutorial(ctx context.Context, issueUUID uuid.UUID, tutorial *domain.TutorialContent, userID string) error {
	_, err := s.db.TutorialContent.Update().
		Where(tutorialcontent.IssueIDEQ(issueUUID)).
		SetTitle(tutorial.Title).
		SetMarkdownBody(tutorial.MarkdownBody).
		SetStatus(tutorialcontent.StatusCompleted).
		Save(ctx)
	if err != nil {
		return err
	}

	// For existing tutorials, link the user if they don't already have access
	if userID != "" {
		content, err := s.db.TutorialContent.Query().
			Where(tutorialcontent.IssueIDEQ(issueUUID)).
			Only(ctx)
		if err == nil {
			// Not the original requester since tutorial already existed
			if err := s.createTutorialAccess(ctx, userID, content.ID, false); err != nil {
				s.logger.Warn("failed to create tutorial access for existing tutorial",
					"user_id", userID,
					"content_id", content.ID,
					"error", err,
				)
			}
		}
	}
	return nil
}

// createTutorialAccess creates a Tutorial row linking user to content.
func (s *Service) createTutorialAccess(ctx context.Context, userIDStr string, contentID uuid.UUID, isOriginalRequester bool) error {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	_, err = s.db.Tutorial.Create().
		SetUserID(userID).
		SetContentID(contentID).
		SetIsOriginalRequester(isOriginalRequester).
		Save(ctx)

	if err != nil {
		// Ignore duplicate key errors - user already has access
		if ent.IsConstraintError(err) {
			return nil
		}
		return err
	}
	return nil
}

// ensureProjectAndIssue ensures the referenced Project and GithubIssue exist.
// TutorialContent has a required FK to GithubIssue via TutorialContent.issue_id -> GithubIssue.id.
func (s *Service) ensureProjectAndIssue(ctx context.Context, payload *transformer.StreamIssuePayload) (uuid.UUID, error) {
	// Upsert-ish: look up by unique full_name first.
	var proj *ent.Project
	proj, err := s.db.Project.Query().
		Where(project.FullNameEQ(payload.FullName)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return uuid.Nil, fmt.Errorf("query project: %w", err)
		}

		proj, err = s.db.Project.Create().
			SetGhRepoID(payload.RepoID).
			SetOwnerHandle(payload.Owner).
			SetRepoName(payload.Repo).
			SetFullName(payload.FullName).
			SetLanguage(payload.RepoLanguage).
			Save(ctx)
		if err != nil {
			// Another worker may have created it (unique full_name / gh_repo_id).
			if ent.IsConstraintError(err) {
				proj, err = s.db.Project.Query().
					Where(project.FullNameEQ(payload.FullName)).
					Only(ctx)
			}
			if err != nil {
				return uuid.Nil, fmt.Errorf("create project: %w", err)
			}
		}
	}

	issueUUID := issueIDToUUID(payload.IssueID)

	// Ensure github issue row exists with deterministic ID that matches tutorial_content.issue_id.
	_, err = s.db.GithubIssue.Query().
		Where(githubissue.IDEQ(issueUUID)).
		Only(ctx)
	if err == nil {
		return issueUUID, nil
	}
	if !ent.IsNotFound(err) {
		return uuid.Nil, fmt.Errorf("query github issue: %w", err)
	}

	raw := map[string]interface{}{
		"gh_issue_id":      payload.IssueID,
		"issue_number":     payload.IssueNumber,
		"owner":            payload.Owner,
		"repo":             payload.Repo,
		"full_name":        payload.FullName,
		"title":            payload.Title,
		"body":             payload.Body,
		"labels":           payload.Labels,
		"state":            payload.State,
		"html_url":         payload.HTMLURL,
		"repo_id":          payload.RepoID,
		"repo_language":    payload.RepoLanguage,
		"repo_stars":       payload.RepoStars,
		"repo_description": payload.RepoDescription,
		"collected_at":     payload.CollectedAt,
		"collected_at_rfc": payload.CollectedAt.Format(time.RFC3339),
	}

	_, err = s.db.GithubIssue.Create().
		SetID(issueUUID).
		SetProjectID(proj.ID).
		SetIssueNumber(payload.IssueNumber).
		SetGhIssueID(payload.IssueID).
		SetRawData(raw).
		Save(ctx)
	if err != nil {
		// If it already exists (race), we're good.
		if ent.IsConstraintError(err) {
			return issueUUID, nil
		}
		return uuid.Nil, fmt.Errorf("create github issue: %w", err)
	}

	return issueUUID, nil
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
// If maxLen is less than 4, returns empty string to avoid panic.
func truncateLog(s string, maxLen int) string {
	if maxLen < 4 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// LLM response validation errors.
var (
	ErrEmptyLLMResponse   = errors.New("ai-processor: LLM returned empty response")
	ErrResponseTooShort   = errors.New("ai-processor: LLM response too short to be valid tutorial")
	ErrResponseTooLong    = errors.New("ai-processor: LLM response exceeds maximum length")
	ErrResponseNoMarkdown = errors.New("ai-processor: LLM response doesn't appear to be valid markdown")
)

// LLM response constraints.
const (
	minResponseLength = 100     // Minimum reasonable tutorial length
	maxResponseLength = 1000000 // 1MB max response
)

// validateLLMResponse checks if the LLM response is valid before parsing.
func validateLLMResponse(content string) error {
	// Check for empty response
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ErrEmptyLLMResponse
	}

	// Check minimum length
	if len(trimmed) < minResponseLength {
		return fmt.Errorf("%w: got %d chars, need at least %d", ErrResponseTooShort, len(trimmed), minResponseLength)
	}

	// Check maximum length
	if len(trimmed) > maxResponseLength {
		return fmt.Errorf("%w: got %d chars, max is %d", ErrResponseTooLong, len(trimmed), maxResponseLength)
	}

	// Check for markdown structure (should have at least one heading)
	if !strings.Contains(trimmed, "#") {
		return ErrResponseNoMarkdown
	}

	return nil
}

// issueIDToUUID converts a GitHub issue ID (int64) to a deterministic UUID.
// Uses UUID v5 (SHA-1 based) with a fixed namespace to ensure:
// - The same issue ID always produces the same UUID
// - The UUID is valid and properly formatted
func issueIDToUUID(issueID int64) uuid.UUID {
	// Convert int64 to bytes for UUID v5 name
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(issueID))
	return uuid.NewSHA1(IssueSightNamespace, b)
}
