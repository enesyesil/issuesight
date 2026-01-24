// Package main is the entry point for the AI Processor service.
//
// The AI Processor service consumes GitHub issue events from Redis Streams
// and generates tutorial content using LLMs, persisting results to PostgreSQL.
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

	"github.com/issuesight/issuesight/backend/ai-processor/handler"
	"github.com/issuesight/issuesight/backend/ai-processor/llm"
	"github.com/issuesight/issuesight/internal/config"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	platformllm "github.com/issuesight/issuesight/internal/platform/llm"
	"github.com/issuesight/issuesight/internal/platform/log"
	"github.com/issuesight/issuesight/internal/platform/redis"
	"github.com/issuesight/issuesight/internal/platform/stream"
)

const (
	serviceName     = "ai-processor"
	defaultPort     = "8082"
	shutdownTimeout = 30 * time.Second
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Use ai-processor-specific port if PORT not set
	port := cfg.Port
	if port == "8080" {
		port = defaultPort
	}

	// 2. Initialize logger
	logger := log.New(log.Config{
		Level:   cfg.LogLevel,
		Format:  cfg.LogFormat,
		Service: serviceName,
	})
	log.SetDefault(logger)

	log.Info("starting ai-processor service",
		"port", port,
		"environment", cfg.Environment,
		"log_level", cfg.LogLevel,
		"llm_provider", cfg.LLMProvider,
	)

	// 3. Validate LLM API key is configured (unless using Ollama)
	if cfg.LLMProvider != "ollama" && cfg.LLMAPIKey == "" {
		log.Error("LLM_API_KEY environment variable is required")
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

	// 5. Initialize PostgreSQL client (Ent)
	dbClient, err := ent.Open("postgres", cfg.PostgresURL)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	// Test PostgreSQL connection
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if err := dbClient.Schema.Create(ctx); err != nil {
		cancel()
		log.Error("failed to run database migration", "error", err)
		os.Exit(1)
	}
	cancel()
	log.Info("connected to postgres")

	// 6. Initialize LLM client
	platformLLM, err := platformllm.New(platformllm.Config{
		Provider:    cfg.LLMProvider,
		APIKey:      cfg.LLMAPIKey,
		Model:       cfg.LLMModel,
		BaseURL:     cfg.LLMBaseURL,
		Temperature: cfg.LLMTemperature,
		MaxTokens:   cfg.LLMMaxTokens,
	})
	if err != nil {
		log.Error("failed to create LLM client", "error", err)
		os.Exit(1)
	}
	log.Info("initialized LLM client",
		"provider", cfg.LLMProvider,
		"model", cfg.LLMModel,
	)

	// Wrap with retry logic
	llmClient, err := llm.New(platformLLM, llm.Config{
		MaxRetries: 3,
		BaseDelay:  time.Second,
		MaxDelay:   30 * time.Second,
		Logger:     logger,
	})
	if err != nil {
		log.Error("failed to create LLM client wrapper", "error", err)
		os.Exit(1)
	}

	// 7. Initialize Redis consumer
	consumer, err := stream.NewRedisConsumer(redisClient, stream.DefaultConsumerConfig())
	if err != nil {
		log.Error("failed to create stream consumer", "error", err)
		os.Exit(1)
	}

	// 8. Initialize AI processor service
	aiService, err := NewService(ServiceConfig{
		Consumer:  consumer,
		LLMClient: llmClient,
		DB:        dbClient,
		Logger:    logger,
	})
	if err != nil {
		log.Error("failed to create ai processor service", "error", err)
		os.Exit(1)
	}

	// 9. Set up HTTP server (health endpoint)
	mux := http.NewServeMux()

	healthChecker := handler.NewHealthChecker(redisClient, dbClient, true)
	mux.HandleFunc("/health", healthChecker.Health())

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 10. Start HTTP server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 11. Start stream consumer in goroutine
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	consumerErrors := make(chan error, 1)
	go func() {
		log.Info("stream consumer starting")
		if err := aiService.Start(consumerCtx); err != nil && !errors.Is(err, context.Canceled) {
			consumerErrors <- err
		}
	}()

	// 12. Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Error("server error", "error", err)
	case err := <-consumerErrors:
		log.Error("consumer error", "error", err)
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig)
	}

	// 13. Graceful shutdown
	log.Info("shutting down service...")

	// Stop consumer first
	consumerCancel()

	// Shutdown HTTP server
	ctx, cancel = context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}

	log.Info("ai-processor service stopped")
}
