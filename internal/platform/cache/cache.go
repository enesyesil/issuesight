// Package cache provides a simple key-value caching interface.

// WHAT IS CACHING?
// Caching is storing frequently accessed data in a fast location (like RAM)
// so you don't have to fetch it from a slower source (like a database) every time.

// ANALOGY - THE LIBRARY DESK:
// Imagine you're studying and need to look things up:
//   - Slow way: Walk to the bookshelf, find the book, bring it back (database query)
//   - Fast way: Keep frequently used books on your desk (cache)

// THE CACHE-ASIDE PATTERN:
// This is how we use caching in IssueSight:
//
//  1. User requests issue #123
//  2. Check cache: "Do we have issue:123?"
//  3. IF YES (cache hit):  Return cached data immediately (~1ms)
//  4. IF NO (cache miss):  Fetch from database (~50ms)
//     Store in cache for next time
//     Return the data


// WHY USE TTL (Time To Live)?
// Data becomes stale. If someone updates issue #123, the cached version
// is now outdated. TTL ensures cached data is refreshed periodically.
// You choose TTL based on how stale your data can be:
//   - 5 minutes: Good for mostly-static data
//   - 30 seconds: Good for frequently-changing data
//   - 0 (no expiry): Good for immutable data (but use carefully!)
package cache

import (
	"context"
	"errors"
	"time"
)

// =============================================================================
// ERRORS
// =============================================================================

// These errors are returned for various cache operations.
// Using typed errors lets you check specifically what went wrong:
//
//	if errors.Is(err, cache.ErrCacheMiss) {
//	    // Not in cache - fetch from database
//	}
var (
	// ErrCacheMiss is returned when you try to GET a key that doesn't exist.
	//
	// This is EXPECTED behavior, not really an "error"!
	// It just means the data isn't cached, so you need to fetch it from
	// the database and maybe cache it for next time.
	ErrCacheMiss = errors.New("cache: cache miss")

	// ErrNilClient is returned when a nil Redis client is provided.
	ErrNilClient = errors.New("cache: redis client cannot be nil")

	// ErrEmptyKey is returned when the cache key is empty.
	ErrEmptyKey = errors.New("cache: key cannot be empty")

	// ErrNilValue is returned when trying to set a nil value.
	ErrNilValue = errors.New("cache: value cannot be nil")

	// ErrNegativeTTL is returned when TTL is negative.
	// Note: TTL of 0 is valid and means "no expiration".
	ErrNegativeTTL = errors.New("cache: TTL cannot be negative")
)

// =============================================================================
// CACHE INTERFACE
// =============================================================================

// Cache is the interface for key-value caching operations.
//
// All keys are strings, and all values are []byte (raw bytes).
// This is flexible because you can store anything:
//   - JSON:   json.Marshal(myStruct) -> []byte
//   - String: []byte("hello world")
//   - Binary: Raw bytes from anywhere
//
// COMMON OPERATIONS:
//   - Get/Set:     Basic read/write
//   - Delete:      Remove a key (e.g., when data is updated)
//   - MGet/MSet:   Batch operations (more efficient than individual calls)
//   - SetNX:       Set only if key doesn't exist (useful for deduplication)
type Cache interface {
	// Get retrieves a value from the cache.
	//
	// Returns ErrCacheMiss if the key doesn't exist (this is normal!).
	// Returns ErrEmptyKey if key is empty.
	//
	// EXAMPLE:
	//   data, err := cache.Get(ctx, "user:123")
	//   if errors.Is(err, cache.ErrCacheMiss) {
	//       // Handle cache miss
	//   }
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with an expiration time.
	//
	// Parameters:
	//   - key:   Unique identifier for the data (cannot be empty)
	//   - value: The data to store (cannot be nil, but can be empty []byte{})
	//   - ttl:   How long until the key expires (0 = never expire, cannot be negative)
	//
	// VALIDATION:
	//   - key cannot be empty (returns ErrEmptyKey)
	//   - value cannot be nil (returns ErrNilValue)
	//   - ttl cannot be negative (returns ErrNegativeTTL)
	//
	// CHOOSING A TTL:
	//   - Too short: More cache misses, more database load
	//   - Too long:  Data becomes stale
	//   - Just right: Depends on your use case!
	//
	// EXAMPLE:
	//   data, _ := json.Marshal(user)
	//   cache.Set(ctx, "user:123", data, 5*time.Minute)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key from the cache.
	//
	// WHEN TO USE:
	// When data changes and you want to invalidate the cached version.
	// Next time someone requests it, they'll get fresh data from the database.
	//
	// VALIDATION:
	//   - key cannot be empty (returns ErrEmptyKey)
	//
	// EXAMPLE:
	//   // User updated their profile
	//   db.UpdateUser(user)
	//   cache.Delete(ctx, "user:"+user.ID)  // Invalidate cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key is in the cache without retrieving it.
	//
	// WHEN TO USE:
	// When you only need to know IF data exists, not WHAT it is.
	// Slightly more efficient than Get when you don't need the value.
	//
	// VALIDATION:
	//   - key cannot be empty (returns ErrEmptyKey)
	Exists(ctx context.Context, key string) (bool, error)

	// SetNX sets a value ONLY if the key doesn't already exist.
	// "NX" stands for "Not eXists".
	//
	// Returns:
	//   - true:  Key was set (it didn't exist before)
	//   - false: Key already exists (nothing was changed)
	//
	// VALIDATION:
	//   - key cannot be empty (returns ErrEmptyKey)
	//   - value cannot be nil (returns ErrNilValue)
	//   - ttl cannot be negative (returns ErrNegativeTTL)
	//
	// USE CASE - Deduplication:
	// You want to process something only once. Multiple workers might
	// try to process the same item. Use SetNX to "claim" it:
	//
	//   claimed, _ := cache.SetNX(ctx, "processed:issue:123", []byte("1"), 24*time.Hour)
	//   if !claimed {
	//       return  // Another worker already processed it
	//   }
	//   processIssue(123)
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)

	// GetSet atomically sets a new value and returns the old value.
	//
	// "Atomic" means no one can interfere between reading and writing.
	//
	// VALIDATION:
	//   - key cannot be empty (returns ErrEmptyKey)
	//   - value cannot be nil (returns ErrNilValue)
	//
	// USE CASE - Counters with history:
	//   old, _ := cache.GetSet(ctx, "daily-count", []byte("0"))
	//   // old contains yesterday's count, new value is reset to 0
	GetSet(ctx context.Context, key string, value []byte) ([]byte, error)

	// Expire sets (or updates) the TTL on an existing key.
	//
	// WHEN TO USE:
	// You want to extend the life of a cached value without re-setting it.
	//
	// VALIDATION:
	//   - key cannot be empty (returns ErrEmptyKey)
	//   - ttl cannot be negative (returns ErrNegativeTTL)
	//
	// EXAMPLE:
	//   // User is still active, extend their session cache
	//   cache.Expire(ctx, "session:abc123", 30*time.Minute)
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// MGet retrieves multiple values in a single call.
	// "M" stands for "Multi".
	//
	// WHY USE THIS?
	// Making 10 individual Get calls means 10 round trips to Redis.
	// MGet does it in ONE round trip - much faster!
	//
	// Returns a map of key -> value. Missing keys are NOT included.
	// Empty keys in the input are skipped.
	//
	// EXAMPLE:
	//   values, _ := cache.MGet(ctx, "user:1", "user:2", "user:3")
	//   // values might be {"user:1": [...], "user:3": [...]}
	//   // (user:2 wasn't in cache)
	MGet(ctx context.Context, keys ...string) (map[string][]byte, error)

	// MSet stores multiple key-value pairs in a single call.
	//
	// Like MGet, this is more efficient than individual Set calls.
	// All keys get the same TTL.
	//
	// VALIDATION:
	//   - Empty keys are skipped
	//   - Nil values are skipped
	//   - ttl cannot be negative (returns ErrNegativeTTL)
	//
	// EXAMPLE:
	//   cache.MSet(ctx, map[string][]byte{
	//       "user:1": user1Data,
	//       "user:2": user2Data,
	//       "user:3": user3Data,
	//   }, 5*time.Minute)
	MSet(ctx context.Context, pairs map[string][]byte, ttl time.Duration) error
}
