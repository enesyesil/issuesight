// Package config handles environment configuration for all services.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
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
	LLMProvider    string  // "openai", "anthropic", "ollama", "gemini", "cohere"
	LLMAPIKey      string  // API key for the provider
	LLMModel       string  // e.g., "gpt-5", "claude-3-opus"
	LLMBaseURL     string  // for Ollama or custom endpoints
	LLMTemperature float64 // 0.0 - 1.0
	LLMMaxTokens   int

	// Server
	Port         string
	Environment  string // "development" or "production"
	BaseURL      string // Backend base URL (e.g., https://api.issuesight.com)
	FrontendURL  string // Frontend URL (e.g., https://issuesight.com)
	CORSOrigins  string // Comma-separated list of allowed CORS origins
	CollectorURL string // Collector service URL (e.g., http://localhost:8081)

	// Rate limit (per IP): requests per window. Set to 0 to disable.
	RateLimitRequests int           // e.g. 60
	RateLimitWindow   time.Duration // e.g. 1*time.Minute
	// Quota (per user, per day): issue submissions. Set to 0 to disable.
	QuotaDailyLimit int

	// Logging
	LogLevel  string // DEBUG, INFO, WARN, ERROR
	LogFormat string // json, text

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// Concepts
	ConceptsAutoSeed      bool
	ConceptsForceBackfill bool
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		// Database
		PostgresURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/issuesight?sslmode=disable"),

		// Redis
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		// GitHub
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubToken:        getEnv("GITHUB_TOKEN", ""),

		// Google
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),

		// LLM
		LLMProvider:    getEnv("LLM_PROVIDER", "openai"),
		LLMAPIKey:      getEnv("LLM_API_KEY", ""),
		LLMModel:       getEnv("LLM_MODEL", "gpt-5"),
		LLMBaseURL:     getEnv("LLM_BASE_URL", ""),
		LLMTemperature: getEnvFloat("LLM_TEMPERATURE", 0.7),
		LLMMaxTokens:   getEnvInt("LLM_MAX_TOKENS", 4096),

		// Server
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("ENV", "development"),
		BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
		FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:3000"),
		CORSOrigins:  getEnv("CORS_ORIGINS", "http://localhost:3000"),
		CollectorURL: getEnv("COLLECTOR_URL", "http://localhost:8081"),

		// Rate limit: 0 = disabled (use for testing)
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 60),
		RateLimitWindow:   getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		// Quota: 0 = unlimited (use for testing)
		QuotaDailyLimit: getEnvInt("QUOTA_DAILY_LIMIT", 10),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "INFO"),
		LogFormat: getEnv("LOG_FORMAT", "text"),

		// JWT
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiration: getEnvDuration("JWT_EXPIRATION", 24*time.Hour),

		// Concepts
		ConceptsAutoSeed:      getEnvBool("CONCEPTS_AUTO_SEED", false),
		ConceptsForceBackfill: getEnvBool("CONCEPTS_FORCE_BACKFILL", false),
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
	if c.RedisDB < 0 || c.RedisDB > 15 {
		return errors.New("config: REDIS_DB must be between 0 and 15")
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

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
