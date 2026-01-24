package github

import (
	"context"
	"log/slog"
	"sync"
	"time"

	ghlib "github.com/google/go-github/v60/github"
)

// RateLimitObserver tracks GitHub API rate limit status.
//
// GitHub has two types of rate limits:
//  1. Primary: 5000 requests/hour for authenticated users
//  2. Secondary: Abuse detection (triggers 403 with Retry-After)
//
// This observer helps us:
//   - Log rate limit status for monitoring
//   - Wait proactively when limits are low
//   - Handle rate limit errors gracefully
type RateLimitObserver struct {
	mu sync.RWMutex

	// Core rate limit info (from X-RateLimit-* headers)
	Limit     int       // max requests per window
	Remaining int       // requests remaining in current window
	Reset     time.Time // when the limit resets

	// Thresholds
	warningThreshold int // log warning when remaining < this
	pauseThreshold   int // pause requests when remaining < this
}

// NewRateLimitObserver creates a new observer with default thresholds.
func NewRateLimitObserver() *RateLimitObserver {
	return &RateLimitObserver{
		warningThreshold: 100,
		pauseThreshold:   10,
		Limit:            5000, // GitHub default for authenticated
		Remaining:        5000,
		Reset:            time.Now().Add(time.Hour),
	}
}

// UpdateFromResponse extracts rate limit info from a GitHub API response.
func (o *RateLimitObserver) UpdateFromResponse(resp *ghlib.Response) {
	if resp == nil || resp.Rate.Limit == 0 {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.Limit = resp.Rate.Limit
	o.Remaining = resp.Rate.Remaining
	o.Reset = resp.Rate.Reset.Time
}

// ShouldWait returns true if we should pause before making more requests.
func (o *RateLimitObserver) ShouldWait() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Remaining < o.pauseThreshold
}

// WaitDuration returns how long to wait before the rate limit resets.
// Returns 0 if no wait is needed.
func (o *RateLimitObserver) WaitDuration() time.Duration {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.Remaining >= o.pauseThreshold {
		return 0
	}

	waitTime := time.Until(o.Reset)
	if waitTime < 0 {
		return 0
	}

	// Add 1 second buffer
	return waitTime + time.Second
}

// WaitIfNeeded blocks until it's safe to make requests.
// Returns ctx.Err() if the context is cancelled while waiting.
func (o *RateLimitObserver) WaitIfNeeded(ctx context.Context, logger *slog.Logger) error {
	waitDuration := o.WaitDuration()
	if waitDuration == 0 {
		return nil
	}

	if logger != nil {
		logger.Warn("rate limit low, waiting",
			"remaining", o.Remaining,
			"reset_at", o.Reset,
			"wait_seconds", waitDuration.Seconds(),
		)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
		return nil
	}
}

// LogStatus logs the current rate limit status.
func (o *RateLimitObserver) LogStatus(logger *slog.Logger) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	level := slog.LevelDebug
	if o.Remaining < o.warningThreshold {
		level = slog.LevelWarn
	}

	logger.Log(context.Background(), level, "github rate limit status",
		"remaining", o.Remaining,
		"limit", o.Limit,
		"reset_at", o.Reset,
	)
}

// Status returns a snapshot of the current rate limit status.
func (o *RateLimitObserver) Status() (remaining, limit int, reset time.Time) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.Remaining, o.Limit, o.Reset
}

// RetryAfter calculates wait time for a rate limit error response.
// If retryAfter header is available, it uses that; otherwise falls back to reset time.
func (o *RateLimitObserver) RetryAfter(resp *ghlib.Response) time.Duration {
	// Check for Retry-After header (secondary rate limit)
	if resp != nil && resp.Response != nil {
		if retryAfter := resp.Response.Header.Get("Retry-After"); retryAfter != "" {
			// Retry-After can be seconds or a date
			if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
				return seconds
			}
		}
	}

	// Fall back to reset time
	return o.WaitDuration()
}

// ExponentialBackoff calculates backoff duration for retry attempts.
// Used for 5xx errors and secondary rate limits.
func ExponentialBackoff(attempt int) time.Duration {
	base := time.Second
	max := 60 * time.Second

	backoff := base * (1 << uint(attempt)) // 1s, 2s, 4s, 8s, 16s, 32s, 64s...
	if backoff > max {
		backoff = max
	}

	return backoff
}
