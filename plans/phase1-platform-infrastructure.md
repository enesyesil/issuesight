# Phase 1: Platform Infrastructure

## Status: ✅ COMPLETED

This phase implements the foundational platform packages that all services depend on.

## Overview

The platform layer (`internal/platform/`) provides reusable infrastructure abstractions for:
- Logging
- Redis operations (cache, locks, streams)
- Database client
- LLM client wrapper

## Implemented Packages

### 1. Logging (`internal/platform/log/`)

**Status:** ✅ Complete

Structured logging using Go's `log/slog` with:
- JSON and text output formats
- Configurable log levels (DEBUG, INFO, WARN, ERROR)
- Service name injection
- Sensitive data redaction
- Context-based request ID propagation

**Files:**
- `log.go` - Config struct and logger initialization
- `handler.go` - JSON/text handlers
- `context.go` - Request ID propagation
- `redact.go` - Secret redaction utilities
- `log_test.go` - Comprehensive tests

**Usage:**
```go
logger := log.New(log.Config{
    Level:   "INFO",
    Format:  "json",
    Service: "collector",
})
log.SetDefault(logger)
```

---

### 2. Redis Client (`internal/platform/redis/`)

**Status:** ✅ Complete

Redis client initialization with connection pooling and configuration.

**Features:**
- Connection pool management
- Configurable timeouts (dial, read, write)
- Retry logic for transient failures
- Validation (addr, DB range)

**Files:**
- `redis.go` - Client initialization and configuration

**Usage:**
```go
client, err := redis.NewClient(redis.Config{
    Addr:         "localhost:6379",
    Password:     os.Getenv("REDIS_PASSWORD"),
    PoolSize:     100,
    MinIdleConns: 10,
})
defer redis.Close(client)
```

---

### 3. Redis Streams (`internal/platform/stream/`)

**Status:** ✅ Complete

Message streaming using Redis Streams for service-to-service communication.

**Features:**
- Publisher with stream trimming (MaxLen)
- Consumer with consumer groups
- Message acknowledgment
- Error handling

**Files:**
- `stream.go` - Interface definitions
- `redis.go` - Redis Streams implementation

**Usage:**
```go
// Publishing
publisher, _ := stream.NewRedisPublisher(redisClient, stream.PublisherConfig{
    MaxLen: 10000,
})
publisher.Publish(ctx, "github-events", map[string]interface{}{
    "repo": "owner/repo",
    "issue": "123",
})

// Consuming
consumer, _ := stream.NewRedisConsumer(redisClient, stream.DefaultConsumerConfig())
consumer.Consume(ctx, "github-events", "ai-workers", "worker-1", func(msg stream.Message) error {
    // Process message
    return nil
})
```

---

### 4. Distributed Locking (`internal/platform/lock/`)

**Status:** ✅ Complete

Distributed locking to prevent duplicate processing across service instances.

**Features:**
- Lock acquisition with TTL
- Lock extension
- Retry logic
- Automatic release on context cancellation

**Files:**
- `lock.go` - Interface definitions
- `redis.go` - Redis-based implementation

**Usage:**
```go
locker, _ := lock.NewRedisLocker(redisClient)
lock, err := locker.Acquire(ctx, "issue:123", 30*time.Second)
defer lock.Release(ctx)
```

---

### 5. Cache (`internal/platform/cache/`)

**Status:** ✅ Complete

Cache-aside pattern for high-speed read access.

**Features:**
- Set/Get operations
- Batch operations (MGet, MSet)
- TTL support
- Cache miss detection

**Files:**
- `cache.go` - Interface definitions
- `redis.go` - Redis-based implementation

**Usage:**
```go
cache, _ := cache.NewRedisCache(redisClient)
cache.Set(ctx, "issue:123", jsonData, 5*time.Minute)
data, err := cache.Get(ctx, "issue:123")
if errors.Is(err, cache.ErrCacheMiss) {
    // Fetch from database
}
```

---

### 6. Database Client (`internal/platform/db/`)

**Status:** ✅ Complete

Database client using Ent ORM with PostgreSQL.

**Features:**
- Ent schema definitions
- Generated type-safe queries
- Migration support
- Transaction handling

**Files:**
- `db.go` - Client initialization
- `ent/` - Generated Ent code and schemas

**Schemas:**
- `user.go` - User accounts
- `user_identity.go` - OAuth provider mappings
- `project.go` - GitHub repositories
- `github_issue.go` - GitHub issues
- `tutorial.go` - Tutorial access records
- `tutorial_content.go` - AI-generated content
- `concept.go` - Reusable concepts
- `concept_relationship.go` - Concept hierarchies
- `project_concept.go` - Project-concept mappings
- `tutorial_concept.go` - Tutorial-concept mappings

---

### 7. LLM Client (`internal/platform/llm/`)

**Status:** ✅ Complete

LLM client wrapper for AI content generation.

**Files:**
- `llm.go` - LLM interface and client
- `langchain.go` - LangChain integration (if applicable)

---

## Key Design Decisions

### Error Handling

All packages use typed errors for clean error handling:
- `cache.ErrCacheMiss`
- `lock.ErrLockNotAcquired`
- `stream.ErrEmptyStreamName`
- `redis.ErrEmptyAddr`

### Key Prefixes

Redis implementations use key prefixes to avoid collisions:
- Cache: `cache:`
- Locks: `lock:`
- Streams: (no prefix, stream names are unique)

### Shared Redis Instance

All three abstractions (stream, lock, cache) share the same Redis instance but use different data structures and key namespaces.

---

## Testing

All packages include comprehensive unit tests:
- Table-driven tests for edge cases
- Mock implementations where applicable
- Error path testing
- Validation testing

---

## Next Steps

With the platform layer complete, services can now:
1. Use structured logging consistently
2. Connect to Redis for caching, locking, and streaming
3. Access the database through Ent
4. Call LLM providers for AI features

**See:** [Phase 3: Collector Service](./phase3-collector-service.md) for first service implementation.
