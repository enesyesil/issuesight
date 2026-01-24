// Package main is the entry point for the Collector service.
//
// The Collector service fetches GitHub issues and publishes them
// to Redis Streams for AI processing.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/issuesight/issuesight/backend/collector/github"
	"github.com/issuesight/issuesight/backend/collector/handler"
	"github.com/issuesight/issuesight/internal/config"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/lock"
	"github.com/issuesight/issuesight/internal/platform/log"
	"github.com/issuesight/issuesight/internal/platform/redis"
	"github.com/issuesight/issuesight/internal/platform/stream"
)

const (
	serviceName     = "collector"
	defaultPort     = "8081"
	shutdownTimeout = 30 * time.Second
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Can't use structured logging yet, fall back to stdlib
		panic("failed to load config: " + err.Error())
	}

	// Use collector-specific port if PORT not set
	port := cfg.Port
	if port == "8080" { // Default from config, use collector default
		port = defaultPort
	}

	// 2. Initialize logger
	logger := log.New(log.Config{
		Level:   cfg.LogLevel,
		Format:  cfg.LogFormat,
		Service: serviceName,
	})
	log.SetDefault(logger)

	log.Info("starting collector service",
		"port", port,
		"environment", cfg.Environment,
		"log_level", cfg.LogLevel,
	)

	// 3. Validate GitHub token is configured
	if cfg.GitHubToken == "" {
		log.Error("GITHUB_TOKEN environment variable is required")
		os.Exit(1)
	}

	// 4. Initialize Redis client
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

	// 5. Initialize Redis publisher
	publisher, err := stream.NewRedisPublisher(redisClient, stream.PublisherConfig{
		MaxLen: domain.DefaultStreamMaxLen,
	})
	if err != nil {
		log.Error("failed to create stream publisher", "error", err)
		os.Exit(1)
	}

	// 6. Initialize distributed locker
	locker, err := lock.NewRedisLocker(redisClient)
	if err != nil {
		log.Error("failed to create locker", "error", err)
		os.Exit(1)
	}

	// 7. Initialize GitHub client
	ghClient, err := github.NewClient(github.ClientConfig{
		Token:  cfg.GitHubToken,
		Logger: logger,
	})
	if err != nil {
		log.Error("failed to create github client", "error", err)
		os.Exit(1)
	}

	// Validate GitHub token
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	if err := ghClient.ValidateToken(ctx); err != nil {
		cancel()
		log.Error("github token validation failed", "error", err)
		os.Exit(1)
	}
	cancel()

	// 8. Initialize collector service
	collectorService, err := NewService(ServiceConfig{
		GitHub:     ghClient,
		Publisher:  publisher,
		Locker:     locker,
		Logger:     logger,
		StreamName: domain.StreamGitHubEvents,
		LockTTL:    domain.DefaultLockTTL,
	})
	if err != nil {
		log.Error("failed to create collector service", "error", err)
		os.Exit(1)
	}

	// 9. Set up HTTP server
	mux := http.NewServeMux()

	// Health endpoint
	healthChecker := handler.NewHealthChecker(redisClient, ghClient)
	mux.HandleFunc("/health", healthChecker.Health())

	// Collect endpoint
	collectHandler := handler.NewCollectHandler(collectorService)
	mux.HandleFunc("/api/collect", collectHandler.Collect())

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 10. Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 11. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Error("server error", "error", err)
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig)
	}

	// 12. Graceful shutdown
	log.Info("shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}

	log.Info("collector service stopped")
}
