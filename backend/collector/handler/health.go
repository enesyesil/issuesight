package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/issuesight/issuesight/backend/collector/github"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// HealthChecker holds dependencies for health checks.
type HealthChecker struct {
	redis  *redis.Client
	github *github.Client
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(redisClient *redis.Client, githubClient *github.Client) *HealthChecker {
	return &HealthChecker{
		redis:  redisClient,
		github: githubClient,
	}
}

// Health returns an HTTP handler for the health endpoint.
func (h *HealthChecker) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		checks := make(map[string]string)
		healthy := true

		// Check Redis
		if err := h.checkRedis(ctx); err != nil {
			checks["redis"] = "failed: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}

		// Check GitHub (rate limit endpoint validates token)
		if err := h.checkGitHub(ctx); err != nil {
			checks["github"] = "failed: " + err.Error()
			healthy = false
		} else {
			checks["github"] = "ok"
		}

		status := "healthy"
		statusCode := http.StatusOK
		if !healthy {
			status = "unhealthy"
			statusCode = http.StatusServiceUnavailable
		}

		response := HealthResponse{
			Status: status,
			Checks: checks,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// checkRedis pings Redis to verify connectivity.
func (h *HealthChecker) checkRedis(ctx context.Context) error {
	if h.redis == nil {
		return nil // Skip if not configured
	}
	return h.redis.Ping(ctx).Err()
}

// checkGitHub validates the GitHub token is still valid.
func (h *HealthChecker) checkGitHub(ctx context.Context) error {
	if h.github == nil {
		return nil // Skip if not configured
	}
	return h.github.ValidateToken(ctx)
}
