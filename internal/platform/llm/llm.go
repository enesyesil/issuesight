// Package llm provides a provider-agnostic interface for LLM operations.
package llm

import (
	"context"
	"errors"
)

// Errors
var (
	ErrNilClient       = errors.New("llm: client cannot be nil")
	ErrEmptyPrompt     = errors.New("llm: prompt cannot be empty")
	ErrInvalidProvider = errors.New("llm: invalid provider")
	ErrGeneration      = errors.New("llm: generation failed")
)

// Config holds LLM provider configuration.
type Config struct {
	Provider    string  // openai, anthropic, ollama, gemini
	APIKey      string  // API key for the provider
	Model       string  // model name (e.g., gpt-4o, claude-3-opus)
	BaseURL     string  // for Ollama or custom endpoints
	Temperature float64 // 0.0 - 1.0
	MaxTokens   int     // max response tokens
}

// Validate checks that required fields are set.
func (c Config) Validate() error {
	if c.Provider == "" {
		return errors.New("llm: provider is required")
	}
	if c.Provider != "ollama" && c.APIKey == "" {
		return errors.New("llm: api key is required for " + c.Provider)
	}
	if c.Model == "" {
		return errors.New("llm: model is required")
	}
	if c.Temperature < 0 || c.Temperature > 1 {
		return errors.New("llm: temperature must be between 0 and 1")
	}
	return nil
}

// Generator is the interface for generating text from prompts.
type Generator interface {
	// Generate produces a text response from a prompt.
	Generate(ctx context.Context, prompt string) (string, error)

	// GenerateWithSystem produces a response using system + user prompts.
	GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
