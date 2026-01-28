// Package llm provides an LLM client wrapper with retry logic for the AI Processor.

// It wraps the platform LLM client (internal/platform/llm) and adds:
//   - Retry logic with exponential backoff
//   - Rate limit handling
//   - Response validation
package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	platformllm "github.com/issuesight/issuesight/internal/platform/llm"
)

// Errors for LLM operations.
var (
	ErrNilClient          = errors.New("llm: client cannot be nil")
	ErrEmptyResponse      = errors.New("llm: received empty response from LLM")
	ErrMaxRetriesExceeded = errors.New("llm: max retries exceeded")
	ErrInvalidAPIKey      = errors.New("llm: invalid API key")
)

// Config holds configuration for the LLM client wrapper.
type Config struct {
	// MaxRetries is the maximum number of retry attempts for transient errors. Default: 3
	MaxRetries int

	// BaseDelay is the initial delay between retries.
	// Subsequent retries use exponential backoff: baseDelay * 2^attempt Default: 1 second
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	// Default: 30 seconds
	MaxDelay time.Duration

	// Logger for logging retry attempts and errors.
	Logger *slog.Logger
}

// DefaultConfig returns sensible defaults for the LLM client.
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Logger:     slog.Default(),
	}
}

// Client wraps the platform LLM client with retry logic.
type Client struct {
	inner  *platformllm.Client
	config Config
}

// New creates a new LLM client wrapper.
func New(inner *platformllm.Client, cfg Config) (*Client, error) {
	if inner == nil {
		return nil, ErrNilClient
	}

	// Apply defaults
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 1 * time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Client{
		inner:  inner,
		config: cfg,
	}, nil
}

// Generate produces a response from a prompt with retry logic.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	return c.generateWithRetry(ctx, "", prompt)
}

// GenerateWithSystem produces a response using system and user prompts with retry logic.
func (c *Client) GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.generateWithRetry(ctx, systemPrompt, userPrompt)
}

// generateWithRetry handles the retry logic for LLM calls.
func (c *Client) generateWithRetry(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Check context before attempting
		if err := ctx.Err(); err != nil {
			return "", err
		}

		// Wait before retry (skip for first attempt)
		if attempt > 0 {
			delay := c.calculateDelay(attempt)
			c.config.Logger.Info("retrying LLM call",
				"attempt", attempt,
				"delay", delay,
				"last_error", lastErr,
			)

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		// Make the LLM call
		var response string
		var err error

		if systemPrompt != "" {
			response, err = c.inner.GenerateWithSystem(ctx, systemPrompt, userPrompt)
		} else {
			response, err = c.inner.Generate(ctx, userPrompt)
		}

		if err == nil {
			// Validate response
			if strings.TrimSpace(response) == "" {
				lastErr = ErrEmptyResponse
				continue
			}
			return response, nil
		}

		// Check if error is retryable
		if !c.isRetryable(err) {
			return "", err
		}

		lastErr = err
	}

	return "", fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// calculateDelay returns the delay for the given retry attempt using exponential backoff.
func (c *Client) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^(attempt-1)
	delay := c.config.BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay > c.config.MaxDelay {
			delay = c.config.MaxDelay
			break
		}
	}
	return delay
}

// isRetryable determines if an error should trigger a retry.
func (c *Client) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Rate limits - definitely retryable
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	// Timeouts - retryable
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") {
		return true
	}

	// Network errors - retryable
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "temporary") {
		return true
	}

	// Server errors (5xx) - retryable
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	// Authentication errors - NOT retryable
	if strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "invalid api key") ||
		strings.Contains(errStr, "unauthorized") {
		return false
	}

	// Default: not retryable (fail fast for unknown errors)
	return false
}

// Provider returns the configured LLM provider name.
func (c *Client) Provider() string {
	return c.inner.Provider()
}
