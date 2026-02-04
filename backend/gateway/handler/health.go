// Package handler provides HTTP handlers for the Gateway service.
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

// HealthHandler holds dependencies for health checks.
type HealthHandler struct {
	redis *redis.Client
	db    *ent.Client
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(redisClient *redis.Client, dbClient *ent.Client) *HealthHandler {
	return &HealthHandler{
		redis: redisClient,
		db:    dbClient,
	}
}

// Health returns an HTTP handler for the health endpoint.
// @Summary      Health Check
// @Description  Checks the health of the service and its dependencies.
// @Tags         health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /health [get]
func (h *HealthHandler) Health() http.HandlerFunc {
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
func (h *HealthHandler) checkRedis(ctx context.Context) error {
	if h.redis == nil {
		return nil // Skip if not configured
	}
	return h.redis.Ping(ctx).Err()
}

// checkPostgres verifies PostgreSQL connectivity.
func (h *HealthHandler) checkPostgres(ctx context.Context) error {
	if h.db == nil {
		return nil // Skip if not configured
	}
	// Use a simple query to check connectivity
	_, err := h.db.User.Query().Limit(0).All(ctx)
	return err
}
