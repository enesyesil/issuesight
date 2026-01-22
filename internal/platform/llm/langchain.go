package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// Client wraps a LangChain LLM model.
type Client struct {
	model llms.Model
	cfg   Config
}

// New creates a new LLM client based on the provider in config.
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	model, err := createModel(cfg)
	if err != nil {
		return nil, fmt.Errorf("llm: failed to create model: %w", err)
	}

	return &Client{
		model: model,
		cfg:   cfg,
	}, nil
}

// createModel creates the appropriate LangChain model based on provider.
func createModel(cfg Config) (llms.Model, error) {
	switch cfg.Provider {
	case "openai":
		opts := []openai.Option{
			openai.WithToken(cfg.APIKey),
			openai.WithModel(cfg.Model),
		}
		if cfg.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
		}
		return openai.New(opts...)

	case "anthropic":
		return anthropic.New(
			anthropic.WithToken(cfg.APIKey),
			anthropic.WithModel(cfg.Model),
		)

	case "ollama":
		opts := []ollama.Option{
			ollama.WithModel(cfg.Model),
		}
		if cfg.BaseURL != "" {
			opts = append(opts, ollama.WithServerURL(cfg.BaseURL))
		}
		return ollama.New(opts...)

	case "gemini":
		return googleai.New(
			context.Background(),
			googleai.WithAPIKey(cfg.APIKey),
			googleai.WithDefaultModel(cfg.Model),
		)

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, cfg.Provider)
	}
}

// Generate produces a response from a single prompt.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", ErrEmptyPrompt
	}

	opts := []llms.CallOption{
		llms.WithTemperature(c.cfg.Temperature),
	}
	if c.cfg.MaxTokens > 0 {
		opts = append(opts, llms.WithMaxTokens(c.cfg.MaxTokens))
	}

	response, err := llms.GenerateFromSinglePrompt(ctx, c.model, prompt, opts...)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGeneration, err)
	}

	return response, nil
}

// GenerateWithSystem produces a response using system and user prompts.
func (c *Client) GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if userPrompt == "" {
		return "", ErrEmptyPrompt
	}

	opts := []llms.CallOption{
		llms.WithTemperature(c.cfg.Temperature),
	}
	if c.cfg.MaxTokens > 0 {
		opts = append(opts, llms.WithMaxTokens(c.cfg.MaxTokens))
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	response, err := c.model.GenerateContent(ctx, messages, opts...)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGeneration, err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("%w: no choices returned", ErrGeneration)
	}

	return response.Choices[0].Content, nil
}

// Model returns the underlying LangChain model (for advanced usage).
func (c *Client) Model() llms.Model {
	return c.model
}

// Provider returns the configured provider name.
func (c *Client) Provider() string {
	return c.cfg.Provider
}
