package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/issuesight/issuesight/internal/platform/log"
	"github.com/redis/go-redis/v9"
)

// RateLimiter implements rate limiting using Redis.
type RateLimiter struct {
	redis  *redis.Client
	rate   int           // requests per interval
	window time.Duration // interval duration
}

// NewRateLimiter creates a new rate limiter middleware.
func NewRateLimiter(client *redis.Client, rate int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:  client,
		rate:   rate,
		window: window,
	}
}

// Limit enforces rate limiting based on the remote IP address.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ip := r.RemoteAddr

		// Simple key based on IP
		key := "ratelimit:" + ip

		// Use a sliding window approach with Redis
		// 1. Add current timestamp to sorted set
		now := time.Now().UnixNano()
		windowStart := now - rl.window.Nanoseconds()

		pipe := rl.redis.Pipeline()
		// Remove old entries
		pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
		// Add new entry
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
		// Count entries in window
		countCmd := pipe.ZCard(ctx, key)
		// Set expiration for the key
		pipe.Expire(ctx, key, rl.window)

		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Error("failed to check rate limit", "error", err)
			// Fail open on Redis error to avoid blocking legitimate traffic
			next.ServeHTTP(w, r)
			return
		}

		count := countCmd.Val()
		if count > int64(rl.rate) {
			log.Warn("rate limit exceeded", "ip", ip, "count", count)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
