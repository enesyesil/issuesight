// Package cache - Redis implementation of caching.
//
// This file implements the Cache interface using Redis as the backend.
//
// WHY REDIS FOR CACHING?
//   - It's in-memory: Data is stored in RAM, so reads are ~1ms
//   - It's networked: Multiple servers can share the same cache
//   - It has TTL built-in: Keys can auto-expire
//   - We already use it: For streams and locks, so no extra infrastructure
//
// KEY PREFIX:
// All keys are prefixed with "cache:" to namespace them.
// This makes it easy to:
//   - Identify cache keys in Redis (vs locks, streams, etc.)
//   - Flush all cache keys without affecting other data
//   - Avoid accidental collisions with other key types
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// cacheKeyPrefix is prepended to all cache keys.
	// Example: key="user:123" becomes "cache:user:123" in Redis
	cacheKeyPrefix = "cache:"
)

// =============================================================================
// REDIS CACHE IMPLEMENTATION
// =============================================================================

// RedisCache implements the Cache interface using Redis.
type RedisCache struct {
	// client is our Redis connection (injected via constructor)
	client *redis.Client
}

// NewRedisCache creates a new Redis-backed cache.
//
// VALIDATION:
// Returns ErrNilClient if client is nil.
//
// EXAMPLE:
//
//	redisClient, _ := redis.NewClient(config)
//	cache, err := cache.NewRedisCache(redisClient)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	cache.Set(ctx, "greeting", []byte("hello"), 5*time.Minute)
//	data, _ := cache.Get(ctx, "greeting")
//	fmt.Println(string(data))  // "hello"
func NewRedisCache(client *redis.Client) (*RedisCache, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &RedisCache{client: client}, nil
}

// =============================================================================
// BASIC OPERATIONS
// =============================================================================

// Get retrieves a value from Redis.
//
// WHAT HAPPENS:
//  1. Validates the key
//  2. Adds "cache:" prefix to the key
//  3. Sends GET command to Redis
//  4. If key exists, returns the value as bytes
//  5. If key doesn't exist, returns ErrCacheMiss
//
// VALIDATION:
//   - key cannot be empty (returns ErrEmptyKey)
//
// REDIS COMMAND: GET cache:user:123
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Validate input
	if key == "" {
		return nil, ErrEmptyKey
	}

	cacheKey := cacheKeyPrefix + key

	// .Bytes() converts the result to []byte
	val, err := c.client.Get(ctx, cacheKey).Bytes()
	if err != nil {
		// redis.Nil is a special "error" that means "key not found"
		// We convert it to our own ErrCacheMiss for cleaner handling
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		// Some other error (connection issues, etc.)
		return nil, fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	return val, nil
}

// Set stores a value in Redis with an expiration time.
//
// WHAT HAPPENS:
//  1. Validates inputs
//  2. Adds "cache:" prefix to the key
//  3. Sends SET command with the value and TTL
//  4. Redis stores it and will auto-delete after TTL
//
// VALIDATION:
//   - key cannot be empty (returns ErrEmptyKey)
//   - value cannot be nil (returns ErrNilValue)
//   - ttl cannot be negative (returns ErrNegativeTTL)
//
// REDIS COMMAND: SET cache:user:123 "value" PX 300000
//
//	(PX 300000 = expires in 300000 milliseconds = 5 minutes)
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Validate inputs
	if key == "" {
		return ErrEmptyKey
	}
	if value == nil {
		return ErrNilValue
	}
	if ttl < 0 {
		return ErrNegativeTTL
	}

	cacheKey := cacheKeyPrefix + key

	// Set the key with expiration
	// If ttl is 0, the key never expires (use carefully!)
	if err := c.client.Set(ctx, cacheKey, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key from Redis.
//
// VALIDATION:
//   - key cannot be empty (returns ErrEmptyKey)
//
// REDIS COMMAND: DEL cache:user:123
//
// Note: DEL doesn't error if the key doesn't exist.
// It just returns 0 (number of keys deleted).
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	// Validate input
	if key == "" {
		return ErrEmptyKey
	}

	cacheKey := cacheKeyPrefix + key

	if err := c.client.Del(ctx, cacheKey).Err(); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists in Redis.
//
// VALIDATION:
//   - key cannot be empty (returns ErrEmptyKey)
//
// REDIS COMMAND: EXISTS cache:user:123
//
// Returns 1 if exists, 0 if not. We convert to boolean.
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	// Validate input
	if key == "" {
		return false, ErrEmptyKey
	}

	cacheKey := cacheKeyPrefix + key

	// EXISTS returns the count of keys that exist (0 or 1 for single key)
	count, err := c.client.Exists(ctx, cacheKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of cache key %s: %w", key, err)
	}

	// count > 0 means the key exists
	return count > 0, nil
}

// =============================================================================
// ATOMIC OPERATIONS
// =============================================================================

// SetNX sets a value only if the key doesn't exist.
// "NX" = "Not eXists"
//
// VALIDATION:
//   - key cannot be empty (returns ErrEmptyKey)
//   - value cannot be nil (returns ErrNilValue)
//   - ttl cannot be negative (returns ErrNegativeTTL)
//
// REDIS COMMAND: SET cache:my-key "value" NX PX 300000
//
// Returns true if the key was set, false if it already existed.
func (c *RedisCache) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	// Validate inputs
	if key == "" {
		return false, ErrEmptyKey
	}
	if value == nil {
		return false, ErrNilValue
	}
	if ttl < 0 {
		return false, ErrNegativeTTL
	}

	cacheKey := cacheKeyPrefix + key

	// SetNX returns true if the key was set, false if it already existed
	wasSet, err := c.client.SetNX(ctx, cacheKey, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx cache key %s: %w", key, err)
	}

	return wasSet, nil
}

// GetSet atomically sets a new value and returns the old value.
//
// VALIDATION:
//   - key cannot be empty (returns ErrEmptyKey)
//   - value cannot be nil (returns ErrNilValue)
//
// REDIS COMMAND: GETSET cache:counter "0"
//
// This is atomic - no other command can run between getting and setting.
func (c *RedisCache) GetSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	// Validate inputs
	if key == "" {
		return nil, ErrEmptyKey
	}
	if value == nil {
		return nil, ErrNilValue
	}

	cacheKey := cacheKeyPrefix + key

	oldVal, err := c.client.GetSet(ctx, cacheKey, value).Bytes()
	if err != nil {
		// Key didn't exist before, so there's no "old value"
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("failed to getset cache key %s: %w", key, err)
	}

	return oldVal, nil
}

// Expire sets or updates the TTL on an existing key.
//
// VALIDATION:
//   - key cannot be empty (returns ErrEmptyKey)
//   - ttl cannot be negative (returns ErrNegativeTTL)
//
// REDIS COMMAND: EXPIRE cache:session:abc 1800
//
//	(1800 seconds = 30 minutes)
//
// Note: If the key doesn't exist, this does nothing (no error).
func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	// Validate inputs
	if key == "" {
		return ErrEmptyKey
	}
	if ttl < 0 {
		return ErrNegativeTTL
	}

	cacheKey := cacheKeyPrefix + key

	if err := c.client.Expire(ctx, cacheKey, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set expiration on cache key %s: %w", key, err)
	}

	return nil
}

// =============================================================================
// BATCH OPERATIONS
// =============================================================================

// MGet retrieves multiple values in a single round trip.
//
// REDIS COMMAND: MGET cache:user:1 cache:user:2 cache:user:3
//
// WHY IS THIS FASTER?
// Each Redis command has network overhead (round trip time ~0.5ms).
// Getting 10 keys individually = 10 round trips = ~5ms overhead
// Getting 10 keys with MGET = 1 round trip = ~0.5ms overhead
//
// HANDLING:
//   - Empty keys in input are skipped
//   - Missing keys in Redis are not included in result
func (c *RedisCache) MGet(ctx context.Context, keys ...string) (map[string][]byte, error) {
	// Handle empty input
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Filter out empty keys and add prefix
	validKeys := make([]string, 0, len(keys))
	keyMapping := make(map[int]string) // index -> original key

	for _, key := range keys {
		if key != "" {
			keyMapping[len(validKeys)] = key
			validKeys = append(validKeys, cacheKeyPrefix+key)
		}
	}

	// If all keys were empty, return empty result
	if len(validKeys) == 0 {
		return make(map[string][]byte), nil
	}

	// MGET returns values in the same order as keys were requested
	vals, err := c.client.MGet(ctx, validKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to mget cache keys: %w", err)
	}

	// Build result map, skipping nil values (cache misses)
	result := make(map[string][]byte)
	for i, val := range vals {
		if val != nil {
			// Redis returns strings, we need to convert to []byte
			if strVal, ok := val.(string); ok {
				// Use the ORIGINAL key (without prefix) for the result map
				result[keyMapping[i]] = []byte(strVal)
			}
		}
	}

	return result, nil
}

// MSet stores multiple key-value pairs in a single round trip.
//
// VALIDATION:
//   - Empty keys are skipped
//   - Nil values are skipped
//   - ttl cannot be negative (returns ErrNegativeTTL)
//
// IMPLEMENTATION NOTE:
// Redis MSET doesn't support per-key TTL, so we use a pipeline instead.
// A pipeline batches multiple commands into one round trip.
//
// PIPELINE vs MSET:
//   - MSET: One command, but no TTL support
//   - Pipeline: Multiple SET commands, but still one round trip
//
// We choose pipeline because we need TTL support.
func (c *RedisCache) MSet(ctx context.Context, pairs map[string][]byte, ttl time.Duration) error {
	// Handle empty input
	if len(pairs) == 0 {
		return nil
	}

	// Validate TTL
	if ttl < 0 {
		return ErrNegativeTTL
	}

	// Create a pipeline - this batches commands for efficiency
	// All commands are sent to Redis in ONE round trip
	pipe := c.client.Pipeline()

	// Queue up a SET command for each valid key-value pair
	validCount := 0
	for key, value := range pairs {
		// Skip empty keys and nil values
		if key == "" || value == nil {
			continue
		}
		cacheKey := cacheKeyPrefix + key
		pipe.Set(ctx, cacheKey, value, ttl)
		validCount++
	}

	// If no valid pairs, nothing to do
	if validCount == 0 {
		return nil
	}

	// Execute all queued commands at once
	// This sends them to Redis in a single round trip
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to mset cache keys: %w", err)
	}

	return nil
}
