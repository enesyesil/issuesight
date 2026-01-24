// Package main is the entry point for the Gateway service.
//
// The Gateway service serves the frontend, manages authentication,
// enforces quotas, and routes requests to the appropriate backends.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/issuesight/issuesight/internal/config"
	"github.com/issuesight/issuesight/internal/platform/cache"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/lock"
	"github.com/issuesight/issuesight/internal/platform/log"
	"github.com/issuesight/issuesight/internal/platform/redis"
	"github.com/issuesight/issuesight/internal/platform/stream"
)

const (
	serviceName     = "gateway"
	defaultPort     = "8080"
	shutdownTimeout = 30 * time.Second
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	port := cfg.Port
	if port == "" {
		port = defaultPort
	}

	// 2. Initialize logger
	logger := log.New(log.Config{
		Level:   cfg.LogLevel,
		Format:  cfg.LogFormat,
		Service: serviceName,
	})
	log.SetDefault(logger)

	log.Info("starting gateway service",
		"port", port,
		"environment", cfg.Environment,
		"log_level", cfg.LogLevel,
	)

	// 3. Initialize Redis client
	redisClient, err := redis.NewClient(redis.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err != nil {
		log.Error("failed to create redis client", "error", err)
		os.Exit(1)
	}
	defer redis.Close(redisClient)

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cancel()
		log.Error("failed to connect to redis", "error", err, "addr", cfg.RedisAddr)
		os.Exit(1)
	}
	cancel()
	log.Info("connected to redis", "addr", cfg.RedisAddr)

	// 4. Initialize PostgreSQL client (Ent)
	dbClient, err := ent.Open("postgres", cfg.PostgresURL)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	// Run migrations
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	if err := dbClient.Schema.Create(ctx); err != nil {
		cancel()
		log.Error("failed to run database migration", "error", err)
		os.Exit(1)
	}
	cancel()
	log.Info("connected to postgres")

	// 5. Initialize platform components
	cacheClient, err := cache.NewRedisCache(redisClient)
	if err != nil {
		log.Error("failed to create cache client", "error", err)
		os.Exit(1)
	}

	locker, err := lock.NewRedisLocker(redisClient)
	if err != nil {
		log.Error("failed to create locker", "error", err)
		os.Exit(1)
	}

	publisher, err := stream.NewRedisPublisher(redisClient, stream.PublisherConfig{
		MaxLen: 10000,
	})
	if err != nil {
		log.Error("failed to create stream publisher", "error", err)
		os.Exit(1)
	}

	// 6. Set up HTTP server using server.go
	server := NewServer(
		DefaultServerConfig(port),
		ServerDeps{
			Redis:     redisClient,
			DB:        dbClient,
			Cache:     cacheClient,
			Locker:    locker,
			Publisher: publisher,
		},
	)

	// 7. Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 8. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Error("server error", "error", err)
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig)
	}

	// 9. Graceful shutdown
	log.Info("shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}

	log.Info("gateway service stopped")
}
