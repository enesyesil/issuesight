// Package gateway provides HTTP server setup for the Gateway service.
package main

import (
	"net/http"
	"time"

	"github.com/issuesight/issuesight/backend/gateway/handler"
	"github.com/issuesight/issuesight/backend/gateway/middleware"
	"github.com/issuesight/issuesight/internal/platform/cache"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/lock"
	"github.com/issuesight/issuesight/internal/platform/stream"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/issuesight/issuesight/backend/gateway/docs" // Swagger docs
)

// ServerConfig holds configuration for the HTTP server.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultServerConfig returns sensible defaults.
func DefaultServerConfig(port string) ServerConfig {
	return ServerConfig{
		Port:         port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// ServerDeps holds dependencies for the HTTP server.
type ServerDeps struct {
	Redis     *redis.Client
	DB        *ent.Client
	Cache     cache.Cache
	Locker    lock.Locker
	Publisher stream.Publisher
}

// NewServer creates and configures the HTTP server.
// @title           IssueSight Gateway API
// @version         1.0
// @description     API Gateway for IssueSight application.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewServer(cfg ServerConfig, deps ServerDeps) *http.Server {
	mux := http.NewServeMux()

	// Register routes
	registerRoutes(mux, deps)

	// Apply middleware chain: Recovery -> Logging -> Handler
	var finalHandler http.Handler = mux
	finalHandler = middleware.Logging(finalHandler)
	finalHandler = middleware.Recovery(finalHandler)

	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      finalHandler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}

// registerRoutes sets up all HTTP routes.
func registerRoutes(mux *http.ServeMux, deps ServerDeps) {
	// Health endpoint (no auth required)
	healthHandler := handler.NewHealthHandler(deps.Redis, deps.DB)
	mux.HandleFunc("/health", healthHandler.Health())

	// Issue submission endpoint
	issueHandler := handler.NewIssueHandler(deps.Publisher, deps.Locker, deps.Cache, deps.DB)
	mux.HandleFunc("POST /api/issues", issueHandler.Submit())

	// Tutorial endpoints
	tutorialHandler := handler.NewTutorialHandler(deps.Cache, deps.DB)
	mux.HandleFunc("GET /api/tutorials/{id}", tutorialHandler.Get())
	mux.HandleFunc("GET /api/tutorials", tutorialHandler.List())

	// Auth endpoints
	authHandler := handler.NewAuthHandler(deps.DB)
	mux.HandleFunc("GET /api/auth/github", authHandler.GitHub())
	mux.HandleFunc("GET /api/auth/google", authHandler.Google())
	mux.HandleFunc("GET /api/auth/callback", authHandler.Callback())
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout())
	mux.HandleFunc("GET /api/auth/me", authHandler.Me())

	// Swagger UI
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
}
