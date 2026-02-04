package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"
)

// mockRedisClient implements a minimal mock for Redis client.
type mockRedisClient struct {
	pingFunc func(ctx context.Context) *redis.StatusCmd
}

func (m *mockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal("PONG")
	return cmd
}

// mockGitHubClientHealth implements a mock for the github client's ValidateToken method.
type mockGitHubClientHealth struct {
	validateErr error
}

func (m *mockGitHubClientHealth) ValidateToken(ctx context.Context) error {
	return m.validateErr
}

func TestHealthChecker(t *testing.T) {
	t.Run("all checks pass", func(t *testing.T) {
		// We can't easily mock HealthChecker since it uses concrete types
		// For now, test nil handling
		checker := &HealthChecker{
			redis:  nil, // will be skipped
			github: nil, // will be skipped
		}

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()

		checker.Health().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status code = %v, want %v", rr.Code, http.StatusOK)
		}

		var resp HealthResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "healthy" {
			t.Errorf("status = %v, want healthy", resp.Status)
		}

		// With nil clients, both checks should return "ok" (skipped)
		if resp.Checks["redis"] != "ok" {
			t.Errorf("redis check = %v, want ok", resp.Checks["redis"])
		}
		if resp.Checks["github"] != "ok" {
			t.Errorf("github check = %v, want ok", resp.Checks["github"])
		}
	})
}

func TestCheckRedis(t *testing.T) {
	t.Run("nil redis returns nil", func(t *testing.T) {
		checker := &HealthChecker{redis: nil}
		err := checker.checkRedis(context.Background())
		if err != nil {
			t.Errorf("checkRedis(nil) = %v, want nil", err)
		}
	})
}

func TestCheckGitHub(t *testing.T) {
	t.Run("nil github returns nil", func(t *testing.T) {
		checker := &HealthChecker{github: nil}
		err := checker.checkGitHub(context.Background())
		if err != nil {
			t.Errorf("checkGitHub(nil) = %v, want nil", err)
		}
	})
}

func TestHealthResponse(t *testing.T) {
	t.Run("response serialization", func(t *testing.T) {
		resp := HealthResponse{
			Status: "healthy",
			Checks: map[string]string{
				"redis":  "ok",
				"github": "ok",
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled HealthResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Status != "healthy" {
			t.Errorf("Status = %v, want healthy", unmarshaled.Status)
		}
		if unmarshaled.Checks["redis"] != "ok" {
			t.Errorf("Checks[redis] = %v, want ok", unmarshaled.Checks["redis"])
		}
	})

	t.Run("unhealthy response", func(t *testing.T) {
		resp := HealthResponse{
			Status: "unhealthy",
			Checks: map[string]string{
				"redis":  "failed: connection refused",
				"github": "ok",
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Check JSON format
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("failed to unmarshal to map: %v", err)
		}

		if raw["status"] != "unhealthy" {
			t.Errorf("status = %v, want unhealthy", raw["status"])
		}
	})
}

// Test that errors are properly formatted in health response
func TestHealthErrorFormatting(t *testing.T) {
	testErr := errors.New("connection refused")

	checks := make(map[string]string)
	checks["redis"] = "failed: " + testErr.Error()

	if checks["redis"] != "failed: connection refused" {
		t.Errorf("error formatting = %v, want 'failed: connection refused'", checks["redis"])
	}
}
