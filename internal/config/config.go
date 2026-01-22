// Package config handles environment configuration for all services.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds all environment variables for the application.
type Config struct {
	// Database
	PostgresURL string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// GitHub OAuth
	GitHubClientID     string
	GitHubClientSecret string
	GitHubToken        string // for API calls

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string

	// LLM (provider-agnostic via LangChain)
	LLMProvider    string  // "openai", "anthropic", "ollama", "gemini"
	LLMAPIKey      string  // API key for the provider
	LLMModel       string  // e.g., "gpt-4o", "claude-3-opus"
	LLMBaseURL     string  // for Ollama or custom endpoints
	LLMTemperature float64 // 0.0 - 1.0
	LLMMaxTokens   int

	// Server
	Port        string
	Environment string // "development" or "production"

	// Logging
	LogLevel  string // DEBUG, INFO, WARN, ERROR
	LogFormat string // json, text

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		// Database
		PostgresURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/issuesight?sslmode=disable"),

		// Redis
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		// GitHub
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),

		// Google
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),

		// LLM
		LLMProvider:    getEnv("LLM_PROVIDER", "openai"),
		LLMAPIKey:      os.Getenv("LLM_API_KEY"),
		LLMModel:       getEnv("LLM_MODEL", "gpt-4o"),
		LLMBaseURL:     getEnv("LLM_BASE_URL", ""),
		LLMTemperature: getEnvFloat("LLM_TEMPERATURE", 0.7),
		LLMMaxTokens:   getEnvInt("LLM_MAX_TOKENS", 4096),

		// Server
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENV", "development"),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "INFO"),
		LogFormat: getEnv("LOG_FORMAT", "text"),

		// JWT
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiration: getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.PostgresURL == "" {
		return errors.New("config: DATABASE_URL is required")
	}
	if c.RedisAddr == "" {
		return errors.New("config: REDIS_ADDR is required")
	}
	if c.Environment == "production" {
		if c.JWTSecret == "change-me-in-production" {
			return errors.New("config: JWT_SECRET must be set in production")
		}
	}
	return nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Helper functions

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
