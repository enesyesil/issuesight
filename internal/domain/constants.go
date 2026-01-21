package domain

import "time"

// Stream names for Redis Streams
const (
	StreamGitHubEvents = "github-events" // collector -> ai-processor
	StreamTutorials    = "tutorials"     // ai-processor -> gateway (notifications)
)

// Consumer group names
const (
	GroupAIProcessor = "ai-processor-group"
	GroupGateway     = "gateway-group"
)

// Cache key prefixes
const (
	CacheKeyTutorial = "tutorial:" // tutorial:{id}
	CacheKeyIssue    = "issue:"    // issue:{gh_issue_id}
	CacheKeyUser     = "user:"     // user:{id}
	CacheKeySession  = "session:"  // session:{token}
)

// Lock key prefixes
const (
	LockKeyIssue = "lock:issue:" // lock:issue:{gh_issue_id}
)

// Default values
const (
	DefaultCacheTTL   = 15 * time.Minute
	DefaultLockTTL    = 5 * time.Minute
	DefaultQuotaLimit = 10 // requests per day per user
	DefaultStreamMaxLen = 10000
)

// OAuth providers
const (
	ProviderGitHub = "github"
	ProviderGoogle = "google"
)

// LLM providers (supported by LangChain Go)
const (
	LLMProviderOpenAI    = "openai"
	LLMProviderAnthropic = "anthropic"
	LLMProviderOllama    = "ollama"
	LLMProviderGemini    = "gemini"
)
