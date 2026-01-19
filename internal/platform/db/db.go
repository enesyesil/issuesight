package db

import (
	"context"
	"fmt"

	"github.com/issuesight/issuesight/internal/platform/db/ent"
)

// Config holds the database connection configuration.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// ConnectionString returns a PostgreSQL connection string.
func (c Config) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// NewClient creates a new Ent client connected to PostgreSQL.
func NewClient(cfg Config) (*ent.Client, error) {
	client, err := ent.Open("postgres", cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}
	return client, nil
}

// Migrate runs the auto-migration tool to create/update the database schema.
func Migrate(ctx context.Context, client *ent.Client) error {
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("failed creating schema resources: %w", err)
	}
	return nil
}
