// Package health provides shared health check utilities for all services.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Checker is the interface for health checks.
type Checker interface {
	// Check performs a health check and returns an error if unhealthy.
	Check(ctx context.Context) error
	// Name returns the name of this checker (e.g., "redis", "postgres").
	Name() string
}

// Status represents the health status of a service.
type Status struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// Handler provides HTTP health check endpoints.
type Handler struct {
	checkers []Checker
	timeout  time.Duration
}

// NewHandler creates a new health check handler.
func NewHandler(checkers ...Checker) *Handler {
	return &Handler{
		checkers: checkers,
		timeout:  5 * time.Second,
	}
}

// WithTimeout sets the timeout for health checks.
func (h *Handler) WithTimeout(timeout time.Duration) *Handler {
	h.timeout = timeout
	return h
}

// Health returns an http.HandlerFunc that performs health checks.
func (h *Handler) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
		defer cancel()

		status := Status{
			Timestamp: time.Now().UTC(),
			Status:    "healthy",
			Checks:    make(map[string]string),
		}

		if len(h.checkers) > 0 {
			// Run all checks concurrently
			var wg sync.WaitGroup
			var mu sync.Mutex
			healthy := true

			for _, checker := range h.checkers {
				wg.Add(1)
				go func(c Checker) {
					defer wg.Done()
					err := c.Check(ctx)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						status.Checks[c.Name()] = "unhealthy: " + err.Error()
						healthy = false
					} else {
						status.Checks[c.Name()] = "healthy"
					}
				}(checker)
			}

			wg.Wait()

			if !healthy {
				status.Status = "unhealthy"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if status.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(status)
	}
}

// Liveness returns a simple liveness probe (always returns 200).
func (h *Handler) Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Status{
			Status:    "alive",
			Timestamp: time.Now().UTC(),
		})
	}
}
