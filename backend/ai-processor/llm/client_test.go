package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockLLMClient implements the methods we need for testing
type mockLLMClient struct {
	responses    []string
	errors       []error
	callCount    int
	systemPrompt string
	userPrompt   string
}

func (m *mockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	idx := m.callCount
	m.callCount++
	m.userPrompt = prompt

	if idx < len(m.errors) && m.errors[idx] != nil {
		return "", m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "default response", nil
}

func (m *mockLLMClient) GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	m.systemPrompt = systemPrompt
	return m.Generate(ctx, userPrompt)
}

func (m *mockLLMClient) Provider() string {
	return "mock"
}

func TestClient_RetryOnTransientError(t *testing.T) {
	// This test verifies the retry logic by checking error classification
	client := &Client{
		config: Config{
			MaxRetries: 3,
			BaseDelay:  10 * time.Millisecond,
			MaxDelay:   100 * time.Millisecond,
		},
	}

	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "rate limit error",
			err:         errors.New("rate limit exceeded"),
			shouldRetry: true,
		},
		{
			name:        "429 error",
			err:         errors.New("429 too many requests"),
			shouldRetry: true,
		},
		{
			name:        "timeout error",
			err:         errors.New("request timeout"),
			shouldRetry: true,
		},
		{
			name:        "connection error",
			err:         errors.New("connection refused"),
			shouldRetry: true,
		},
		{
			name:        "503 server error",
			err:         errors.New("503 service unavailable"),
			shouldRetry: true,
		},
		{
			name:        "401 unauthorized",
			err:         errors.New("401 unauthorized"),
			shouldRetry: false,
		},
		{
			name:        "invalid api key",
			err:         errors.New("invalid api key"),
			shouldRetry: false,
		},
		{
			name:        "unknown error",
			err:         errors.New("some unknown error"),
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryable(tt.err)
			if result != tt.shouldRetry {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, result, tt.shouldRetry)
			}
		})
	}
}

func TestClient_CalculateDelay(t *testing.T) {
	client := &Client{
		config: Config{
			BaseDelay: 1 * time.Second,
			MaxDelay:  30 * time.Second,
		},
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
		{6, 30 * time.Second}, // Capped at MaxDelay
		{7, 30 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := client.calculateDelay(tt.attempt)
			if result != tt.expected {
				t.Errorf("calculateDelay(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestNew_NilClient(t *testing.T) {
	_, err := New(nil, DefaultConfig())
	if err != ErrNilClient {
		t.Errorf("New(nil) error = %v, want %v", err, ErrNilClient)
	}
}

func TestNew_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", cfg.MaxDelay)
	}
}
