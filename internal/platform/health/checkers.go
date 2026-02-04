package health

import (
	"context"
	"fmt"

	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/redis/go-redis/v9"
)

// RedisChecker checks Redis connectivity.
type RedisChecker struct {
	client *redis.Client
}

// NewRedisChecker creates a new Redis health checker.
func NewRedisChecker(client *redis.Client) *RedisChecker {
	return &RedisChecker{client: client}
}

// Check pings Redis to verify connectivity.
func (c *RedisChecker) Check(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return c.client.Ping(ctx).Err()
}

// Name returns the checker name.
func (c *RedisChecker) Name() string {
	return "redis"
}

// PostgresChecker checks PostgreSQL connectivity.
type PostgresChecker struct {
	client *ent.Client
}

// NewPostgresChecker creates a new PostgreSQL health checker.
func NewPostgresChecker(client *ent.Client) *PostgresChecker {
	return &PostgresChecker{client: client}
}

// Check queries the database to verify connectivity.
func (c *PostgresChecker) Check(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("database client is nil")
	}
	// Simple query to verify connection
	_, err := c.client.User.Query().Limit(1).All(ctx)
	if err != nil && err.Error() != "ent: user not found" {
		return err
	}
	return nil
}

// Name returns the checker name.
func (c *PostgresChecker) Name() string {
	return "postgres"
}
