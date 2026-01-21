// Package lock provides distributed locking capabilities.

// WHAT IS A DISTRIBUTED LOCK?
// Imagine you have 3 servers running the same code. If all 3 try to process the same GitHub issue at the same time, you'd get duplicate work (or worse, corrupted data). A distributed lock prevents this.

// ANALOGY - THE BATHROOM KEY:
// Think of a coffee shop with one bathroom and one key:
//   - Only one person can hold the key at a time
//   - When you're done, you return the key
//   - If someone takes too long, the staff has a master key (TTL expiration)

// HOW WE USE THIS IN ISSUESIGHT:
// Before processing an issue, the AI Worker tries to acquire a lock:
//   - "lock:issue:12345" - only one worker can hold this at a time
//   - If another worker already has it, we skip (or wait)
//   - When done, we release the lock for the next issue

// WHY REDIS FOR LOCKS?
//   - It's fast (in-memory)
//   - We already have it (for caching and streams)
//   - It has atomic operations (SET NX) perfect for locking
//   - It has TTL (keys expire automatically if we crash)

package lock

import (
	"context"
	"errors"
	"time"
)



// These are the errors you'll encounter when working with locks.
// Using typed errors (instead of just strings) lets you check specifically
// what went wrong using errors.Is(err, lock.ErrLockNotAcquired).

var (
	// ErrLockNotAcquired means someone else already holds the lock.
	// This is EXPECTED behavior - just means you should skip or wait.
	ErrLockNotAcquired = errors.New("lock: lock not acquired")

	// ErrLockNotHeld means you tried to release/extend a lock you don't own.
	// This can happen if:
	//   - The lock expired (TTL ran out)
	//   - Another process stole it (shouldn't happen with proper TTL)
	//   - You already released it
	ErrLockNotHeld = errors.New("lock: lock not held")

	// ErrNilClient is returned when a nil Redis client is provided.
	ErrNilClient = errors.New("lock: redis client cannot be nil")

	// ErrEmptyKey is returned when the lock key is empty.
	ErrEmptyKey = errors.New("lock: key cannot be empty")

	// ErrInvalidTTL is returned when the TTL is zero or negative.
	ErrInvalidTTL = errors.New("lock: TTL must be positive")
)



// Lock represents a lock that YOU currently hold.

// IMPORTANTE: Once you have a Lock, YOU are responsible for releasing it!
// Use defer to ensure you don't forget:

//	defer lock.Release(ctx)  // Always release when done!

type Lock interface {


	Release(ctx context.Context) error

	// Extend pushes back the lock's expiration time.




	Extend(ctx context.Context, ttl time.Duration) error

	// Key returns the full lock key (including the "lock:" prefix).
	// Useful for logging/debugging.
	Key() string
}


// Locker is the interface for acquiring distributed locks.
//
// USAGE PATTERN:
//  1. Try to acquire a lock with Acquire() or TryAcquire()
//  2. If successful, do your work
//  3. Release the lock when done (use defer!)
//
// CHOOSING THE RIGHT METHOD:
//   - TryAcquire: Try once, fail immediately if locked (non-blocking)
//   - Acquire:    Same as TryAcquire (alias for clarity)
//   - AcquireWithRetry: Keep trying for a while before giving up

type Locker interface {
	// Acquire attempts to get a lock with the given key.
	
	// Parameters:
	//   - key: Unique identifier for what you're locking (e.g., "issue:12345")
	//   - ttl: How long before the lock auto-expires (safety mechanism)

	// Returns:
	//   - Lock: The lock object (if successful)
	//   - error: ErrLockNotAcquired if someone else has it
	
	// CHOOSING A TTL:
	//   - Too short: Lock expires while you're still working
	//   - Too long: If you crash, others wait longer than necessary
	//   - Good rule: 2-3x your expected processing time
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)

	// TryAcquire is the same as Acquire - tries once, returns immediately.
	
	// This name makes it clearer that it doesn't wait/block.
	
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)

	// AcquireWithRetry keeps trying to acquire the lock for a while.
	
	
	// When you MUST have the lock and are willing to wait.
	// The function will try 'retries' times, waiting 'retryDelay' between attempts.

	
	// EXAMPLE:
	//   // Try up to 5 times, waiting 1 second between attempts
	//   lock, err := locker.AcquireWithRetry(ctx, "critical-job", 30*time.Second, 5, time.Second)
	
	// CAUTION:
	// Be careful with retries - if many workers retry simultaneously, you might create a "thundering herd" problem.
	AcquireWithRetry(ctx context.Context, key string, ttl time.Duration, retries int, retryDelay time.Duration) (Lock, error)
}
