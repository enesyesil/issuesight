// Package handler provides HTTP handlers for the AI Processor service.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/issuesight/issuesight/internal/platform/db/ent"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// HealthChecker holds dependencies for health checks.
type HealthChecker struct {
	redis    *redis.Client
	db       *ent.Client
	llmReady bool // Simple flag to indicate LLM was initialized
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(redisClient *redis.Client, dbClient *ent.Client, llmReady bool) *HealthChecker {
	return &HealthChecker{
		redis:    redisClient,
		db:       dbClient,
		llmReady: llmReady,
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

		// Check PostgreSQL
		if err := h.checkPostgres(ctx); err != nil {
			checks["postgres"] = "failed: " + err.Error()
			healthy = false
		} else {
			checks["postgres"] = "ok"
		}

		// Check LLM (just report initialized status)
		if h.llmReady {
			checks["llm"] = "ok"
		} else {
			checks["llm"] = "not initialized"
			healthy = false
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

// checkPostgres verifies PostgreSQL connectivity by running a simple query.
func (h *HealthChecker) checkPostgres(ctx context.Context) error {
	if h.db == nil {
		return nil // Skip if not configured
	}
	// Use a simple query to check connectivity
	// Ent doesn't have a direct Ping, so we use a lightweight operation
	_, err := h.db.TutorialContent.Query().Limit(0).All(ctx)
	return err
}
