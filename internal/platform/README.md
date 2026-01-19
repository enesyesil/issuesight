# Platform Layer

This directory contains infrastructure abstractions for IssueSight.

## Packages

### `redis/`
Redis client initialization and configuration.

```go
import "github.com/issuesight/issuesight/internal/platform/redis"

// Basic usage (development)
client, err := redis.NewClient(redis.Config{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
if err != nil {
    log.Fatal(err)
}
defer redis.Close(client)

// Production usage (with connection pool and timeouts)
client, err := redis.NewClient(redis.Config{
    Addr:         "redis.prod.example.com:6379",
    Password:     os.Getenv("REDIS_PASSWORD"),
    DB:           0,
    PoolSize:     100,              // Connection pool size
    MinIdleConns: 10,               // Keep connections warm
    DialTimeout:  5 * time.Second,  // Connection timeout
    ReadTimeout:  3 * time.Second,  // Read timeout
    WriteTimeout: 3 * time.Second,  // Write timeout
    MaxRetries:   3,                // Retry on transient failures
})
```

**Validation:**
- `Addr` cannot be empty
- `DB` must be between 0 and 15

### `stream/`
Message streaming using Redis Streams for the Collector → AI Worker pipeline.

```go
import "github.com/issuesight/issuesight/internal/platform/stream"

// Publishing (with stream trimming)
publisher, err := stream.NewRedisPublisher(redisClient, stream.PublisherConfig{
    MaxLen: 10000,  // Keep only last 10k messages
})
if err != nil {
    log.Fatal(err)
}

msgID, err := publisher.Publish(ctx, "github-events", map[string]interface{}{
    "repo": "owner/repo",
    "issue": "123",
})

// Consuming
consumer, err := stream.NewRedisConsumer(redisClient, stream.DefaultConsumerConfig())
if err != nil {
    log.Fatal(err)
}

err = consumer.CreateGroup(ctx, "github-events", "ai-workers")
err = consumer.Consume(ctx, "github-events", "ai-workers", "worker-1", func(msg stream.Message) error {
    // Process message
    return nil
})
```

**Validation:**
- Client cannot be nil
- Stream name cannot be empty
- Group name cannot be empty
- Consumer name cannot be empty
- Handler function cannot be nil
- Payload cannot be empty

### `lock/`
Distributed locking to prevent duplicate processing.

```go
import "github.com/issuesight/issuesight/internal/platform/lock"

locker, err := lock.NewRedisLocker(redisClient)
if err != nil {
    log.Fatal(err)
}

// Acquire lock
lock, err := locker.Acquire(ctx, "issue:123", 30*time.Second)
if errors.Is(err, lock.ErrLockNotAcquired) {
    // Lock already held by another worker
    return
}
if err != nil {
    return err
}
defer lock.Release(ctx)

// Extend lock if processing takes longer
err = lock.Extend(ctx, 30*time.Second)
```

**With Retries:**
```go
// Retry up to 5 times with 1 second delay
lock, err := locker.AcquireWithRetry(ctx, "issue:123", 30*time.Second, 5, time.Second)
```

**Validation:**
- Client cannot be nil
- Key cannot be empty
- TTL must be positive

### `cache/`
Cache-aside pattern for API reads.

```go
import "github.com/issuesight/issuesight/internal/platform/cache"

cache, err := cache.NewRedisCache(redisClient)
if err != nil {
    log.Fatal(err)
}

// Set cache
err := cache.Set(ctx, "issue:123", jsonData, 5*time.Minute)

// Get cache
data, err := cache.Get(ctx, "issue:123")
if errors.Is(err, cache.ErrCacheMiss) {
    // Cache miss - fetch from database
}

// Batch operations
values, err := cache.MGet(ctx, "issue:123", "issue:456")
err = cache.MSet(ctx, map[string][]byte{
    "issue:123": data1,
    "issue:456": data2,
}, 5*time.Minute)
```

**Validation:**
- Client cannot be nil
- Key cannot be empty
- Value cannot be nil (for Set operations)
- TTL cannot be negative (0 = no expiration)

### `db/`
Database models and client using Ent ORM.

See the schema files in `db/ent/schema/` for entity definitions.

## Error Handling

All packages use typed errors for clean error handling:

```go
import "errors"

// Cache
if errors.Is(err, cache.ErrCacheMiss) { ... }
if errors.Is(err, cache.ErrEmptyKey) { ... }
if errors.Is(err, cache.ErrNilValue) { ... }

// Lock
if errors.Is(err, lock.ErrLockNotAcquired) { ... }
if errors.Is(err, lock.ErrLockNotHeld) { ... }
if errors.Is(err, lock.ErrEmptyKey) { ... }
if errors.Is(err, lock.ErrInvalidTTL) { ... }

// Stream
if errors.Is(err, stream.ErrNilClient) { ... }
if errors.Is(err, stream.ErrEmptyStreamName) { ... }
if errors.Is(err, stream.ErrEmptyGroupName) { ... }

// Redis
if errors.Is(err, redis.ErrEmptyAddr) { ... }
if errors.Is(err, redis.ErrInvalidDB) { ... }
```

## Key Prefixes

All Redis implementations use key prefixes to avoid collisions:

| Package | Prefix | Example |
|---------|--------|---------|
| Cache | `cache:` | `cache:issue:123` |
| Locks | `lock:` | `lock:issue:123` |
| Streams | (none) | `github-events` |

## Architecture

```
┌─────────────────────────────────────────┐
│         Redis Instance (6379)           │
├─────────────────────────────────────────┤
│  DB 0 (KV)                              │
│  ├── cache:issue:123                    │
│  ├── cache:project:456                  │
│  └── lock:issue:123                     │
│                                         │
│  Streams                                │
│  ├── github-events (MaxLen: 10000)      │
│  └── ai-tasks                           │
└─────────────────────────────────────────┘
```

All three abstractions (stream, lock, cache) share the same Redis instance but use different data structures and key namespaces.

## Graceful Shutdown

Always close the Redis client when shutting down:

```go
client, err := redis.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer redis.Close(client)  // Safe even if client is nil
```
