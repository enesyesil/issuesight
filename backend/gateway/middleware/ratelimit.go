package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	httputil "github.com/issuesight/issuesight/internal/platform/http"
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

// getClientIP extracts the real client IP address from the request.
// It checks X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (can contain multiple IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr

	// Remove port if present
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}

	return ip
}

// Limit enforces rate limiting based on the client IP address.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ip := getClientIP(r)

		// Key based on client IP
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
			httputil.WriteError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests. Please slow down.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
