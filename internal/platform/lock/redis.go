// Package lock - Redis implementation of distributed locking.

// HOW REDIS LOCKS WORK:
// We use Redis's SET command with special options:
//   - NX (Not eXists): Only set if the key doesn't exist
//   - EX/PX: Set an expiration time (TTL)

// THE TOKEN PATTERN:
// When we acquire a lock, we store a random UUID as the value.
// This token proves we OWN the lock. When releasing, we check:


// If yes, delete it. If no, someone else has it (don't delete!).

// WHY USE LUA SCRIPTS?
// Redis operations like GET + DEL aren't atomic by default.
// Between GET and DEL, another process could change the value!
// Lua scripts run atomically in Redis - no race conditions.

package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// lockKeyPrefix is prepended to all lock keys.
	// This namespaces our locks and makes them easy to identify in Redis.
	// Example: key="issue:123" becomes "lock:issue:123" in Redis
	lockKeyPrefix = "lock:"
)


// RedisLocker implements the Locker interface using Redis.

// It uses the "SET key value NX PX timeout" pattern, which is the recommended way to implement distributed locks in Redis.
type RedisLocker struct {
	// client is our Redis connection
	client *redis.Client
}

// NewRedisLocker creates a new Redis-backed distributed lock manager.



func NewRedisLocker(client *redis.Client) (*RedisLocker, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &RedisLocker{client: client}, nil
}

// Acquire attempts to get a lock (same as TryAcquire).

func (l *RedisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	return l.TryAcquire(ctx, key, ttl)
}

// TryAcquire attempts to acquire a lock without waiting.

// WHAT HAPPENS:
//  1. Validates inputs (key and TTL)
//  2. Generates a random UUID as our "ownership token"
//  3. Tries to SET the lock key with NX (only if doesn't exist)
//  4. If successful, returns a Lock object
//  5. If key exists (someone else has it), returns ErrLockNotAcquired

func (l *RedisLocker) TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	// Validate inputs
	if key == "" {
		return nil, ErrEmptyKey
	}
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}

	// Build the full key with our prefix
	lockKey := lockKeyPrefix + key

	// Generate a random token  this is how we'll prove we own the lock later.
	// UUID v4 is random enough that collisions are practically impossible.
	token := uuid.New().String()

	// SetNX = SET if Not eXists
	// Returns true if we got the lock, false if someone else has it
	acquired, err := l.client.SetNX(ctx, lockKey, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock %s: %w", key, err)
	}

	// Someone else already has the lock
	if !acquired {
		return nil, ErrLockNotAcquired
	}

	// Success! Return a Lock object that can be used to release/extend
	return &redisLock{
		client: l.client,
		key:    lockKey,
		token:  token,
	}, nil
}

// AcquireWithRetry keeps trying to acquire a lock with configurable retries.

// ALGORITHM:
//  1. Validates inputs
//  2. Tries to acquire the lock
//  3. If successful, returns immediately
//  4. If lock is held by someone else, waits 'retryDelay' and tries again
//  5. Repeats up to 'retries' times
//  6. If all attempts fail, returns the last error

func (l *RedisLocker) AcquireWithRetry(ctx context.Context, key string, ttl time.Duration, retries int, retryDelay time.Duration) (Lock, error) {
	// Validate inputs upfront (fail fast)
	if key == "" {
		return nil, ErrEmptyKey
	}
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}

	var lastErr error

	// Try retries + 1 times (first try + retries)
	for i := 0; i <= retries; i++ {
		lock, err := l.TryAcquire(ctx, key, ttl)
		if err == nil {
			// Got the lock!
			return lock, nil
		}

		lastErr = err

		// If it's not a  lock held by someone else error, don't retry.
		// Could be a connection error, etc. smth else
		if err != ErrLockNotAcquired {
			return nil, err
		}

		// Don't sleep after the last attempt (we're about to return anyway)
		if i < retries {
			// Wait before retrying, but also check if context was cancelled
			select {
			case <-ctx.Done():
				// Context cancelled (shutdown signal)
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				// Delay elapsed, try again
			}
		}
	}


	return nil, lastErr
}


// redisLock represents an acquired lock.

// It stores the token we used to acquire the lock, which we'll need to prove ownership when releasing or extending.
type redisLock struct {
	client *redis.Client
	key    string // Full key including lock: prefix
	token  string // Our ownership token (UUID)
}

// Release gives up the lock using a Lua script for atomicity.



// The Lua script runs as a single atomic operation in Redis. It kinda provides atomicity like CAS. 
func (l *redisLock) Release(ctx context.Context) error {


	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	// Eval runs our Lua script in Redis
	result, err := l.client.Eval(ctx, script, []string{l.key}, l.token).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock %s: %w", l.key, err)
	}

	// result is 0 if we didn't own the lock (someone else had it)
	// Use safe type assertion to avoid panic on unexpected response type
	deleted, ok := result.(int64)
	if !ok {
		return fmt.Errorf("failed to release lock %s: unexpected result type %T", l.key, result)
	}
	if deleted == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Extend resets the lock's TTL (pushes back expiration).


// Your job is taking longer than expected. Instead of letting the lock expire (which would let someone else grab it while you're still working),  you extend it to buy more time.

// ATOMICITY:
// Like Release, we use a Lua script to atomically check ownership before extending. We don't want to extend someone else's lock! Like CAS.
func (l *redisLock) Extend(ctx context.Context, ttl time.Duration) error {
	// Validate TTL
	if ttl <= 0 {
		return ErrInvalidTTL
	}

	// This Lua script atomically:
	//   1. Checks if we still own the lock
	//   2. If yes, sets a new expiration time
	


	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.token, ttl.Milliseconds()).Result()
	if err != nil {
		return fmt.Errorf("failed to extend lock %s: %w", l.key, err)
	}

	// 0 means we don't own the lock anymore
	// Use safe type assertion to avoid panic on unexpected response type
	extended, ok := result.(int64)
	if !ok {
		return fmt.Errorf("failed to extend lock %s: unexpected result type %T", l.key, result)
	}
	if extended == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Key returns the full lock key (including "lock:" prefix).
// Useful for logging and debugging.
func (l *redisLock) Key() string {
	return l.key
}
