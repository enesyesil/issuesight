// Package gateway provides HTTP server setup for the Gateway service.
package main

import (
	"net/http"
	"time"

	"github.com/issuesight/issuesight/backend/gateway/auth"
	"github.com/issuesight/issuesight/backend/gateway/handler"
	"github.com/issuesight/issuesight/backend/gateway/middleware"
	"github.com/issuesight/issuesight/internal/config"
	"github.com/issuesight/issuesight/internal/platform/cache"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/log"
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
	Redis        *redis.Client
	DB           *ent.Client
	Cache        cache.Cache
	CollectorURL string // URL for the collector service
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
	// Load app config for middleware settings
	appCfg, _ := config.Load()

	mux := http.NewServeMux()

	// Register routes
	registerRoutes(mux, deps)

	// Rate limiter: disabled when RATE_LIMIT_REQUESTS <= 0 (e.g. for testing)
	var rateLimiter *middleware.RateLimiter
	if appCfg.RateLimitRequests > 0 && appCfg.RateLimitWindow > 0 {
		rateLimiter = middleware.NewRateLimiter(deps.Redis, appCfg.RateLimitRequests, appCfg.RateLimitWindow)
	}

	// Initialize CSRF Protection with secure cookies in production
	csrf := middleware.NewCSRFWithConfig(middleware.CSRFConfig{
		SecureCookie: appCfg.IsProduction(),
	})

	// Initialize CORS with configured origins
	cors := middleware.NewCORSWithConfig(middleware.CORSConfig{
		AllowedOrigins: appCfg.CORSOrigins,
	})

	// Initialize Security Headers with HSTS in production
	securityHeaders := middleware.NewSecurityHeadersWithConfig(middleware.SecurityConfig{
		EnableHSTS: appCfg.IsProduction(),
	})

	// Apply middleware chain: Recovery -> CORS -> SecurityHeaders -> Logging -> [RateLimit] -> CSRF -> Handler
	// Order matters! Recovery should be outermost, CSRF innermost.
	var finalHandler http.Handler = mux

	// Inner middlewares
	finalHandler = csrf.Protect(finalHandler)
	if rateLimiter != nil {
		finalHandler = rateLimiter.Limit(finalHandler)
	}

	// Outer middlewares
	finalHandler = middleware.Logging(finalHandler)
	finalHandler = securityHeaders.Handler(finalHandler)
	finalHandler = cors.Handler(finalHandler)
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
	appCfg, _ := config.Load()
	authMiddleware := middleware.NewAuthMiddleware(middleware.JWTConfig{
		Secret:     appCfg.JWTSecret,
		Expiration: appCfg.JWTExpiration,
	})
	quotaMiddleware := middleware.NewQuotaMiddleware(deps.DB)
	if appCfg.QuotaDailyLimit > 0 {
		quotaMiddleware = quotaMiddleware.WithLimit(appCfg.QuotaDailyLimit)
	}

	// Health endpoint (no auth required)
	healthHandler := handler.NewHealthHandler(deps.Redis, deps.DB)
	mux.HandleFunc("/health", healthHandler.Health())

	// Issue submission endpoint (auth + quota when QuotaDailyLimit > 0)
	issueHandler := handler.NewIssueHandler(deps.CollectorURL, deps.Cache, deps.DB)
	var issueChain http.Handler = issueHandler.Submit()
	if appCfg.QuotaDailyLimit > 0 {
		issueChain = quotaMiddleware.EnforceQuota(issueChain)
	}
	issueChain = authMiddleware.RequireAuth(issueChain)
	mux.Handle("POST /api/issues", issueChain)

	// Quota endpoint (auth only); 0 = unlimited
	quotaHandler := handler.NewQuotaHandler(deps.DB).WithLimit(appCfg.QuotaDailyLimit)
	mux.Handle("GET /api/quota", authMiddleware.RequireAuth(quotaHandler.Get()))

	// Tutorial endpoints
	tutorialHandler := handler.NewTutorialHandler(deps.Cache, deps.DB)
	mux.Handle("GET /api/tutorials/{id}", authMiddleware.RequireAuth(tutorialHandler.Get()))
	mux.Handle("GET /api/tutorials", authMiddleware.RequireAuth(tutorialHandler.List()))

	// Concept endpoints (public)
	conceptHandler := handler.NewConceptHandler(deps.DB)
	mux.Handle("GET /api/concepts", conceptHandler.List())
	mux.Handle("GET /api/concepts/{slug}", conceptHandler.Get())

	// Create Redis state store for OAuth (production-ready)
	stateStore, err := auth.NewRedisStateStore(deps.Redis)
	if err != nil {
		log.Warn("failed to create Redis state store, falling back to in-memory", "error", err)
		stateStore = nil // Will use in-memory fallback
	}

	// Auth endpoints
	authHandler := handler.NewAuthHandler(handler.AuthHandlerConfig{
		DB:         deps.DB,
		StateStore: stateStore,
	})
	mux.HandleFunc("GET /api/auth/github", authHandler.GitHub())
	mux.HandleFunc("GET /api/auth/google", authHandler.Google())
	mux.HandleFunc("GET /api/auth/callback", authHandler.Callback())
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout())
	mux.HandleFunc("GET /api/auth/me", authHandler.Me())

	// Swagger UI
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
}
