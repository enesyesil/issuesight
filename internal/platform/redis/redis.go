// Package redis provides a simple way to connect to a Redis server.
//
// WHAT IS REDIS?
// Redis is an in-memory data store - think of it like a super-fast dictionary
// that lives in RAM instead of on disk. It's commonly used for:
//   - Caching: Store frequently accessed data to avoid hitting slower databases
//   - Message queues: Pass messages between different parts of your application
//   - Locks: Coordinate between multiple servers so they don't step on each other
//
// WHY USE THIS PACKAGE?
// This package wraps the official Redis client library and provides a simple
// Config struct so you can easily connect to Redis from anywhere in the app.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// ERRORS
// =============================================================================

// These errors are returned when configuration is invalid.
// Using typed errors lets you check specifically what went wrong:
//
//	if errors.Is(err, redis.ErrEmptyAddr) {
//	    log.Fatal("Redis address not configured!")
//	}
var (
	// ErrEmptyAddr is returned when Config.Addr is empty.
	ErrEmptyAddr = errors.New("redis: address cannot be empty")

	// ErrInvalidDB is returned when Config.DB is outside the valid range (0-15).
	ErrInvalidDB = errors.New("redis: database number must be between 0 and 15")
)

// =============================================================================
// CONFIGURATION
// =============================================================================

// Config holds everything needed to connect to a Redis server.
//
// Think of this like the connection info you'd put in a database URL:
//   - Addr:     Where Redis is running (hostname:port)
//   - Password: The secret password (empty string if Redis has no password)
//   - DB:       Redis has 16 separate databases (0-15), like separate folders
//
// OPTIONAL FIELDS (for production tuning):
//   - PoolSize:     How many connections to keep open
//   - DialTimeout:  How long to wait when connecting
//   - ReadTimeout:  How long to wait for a response
//   - WriteTimeout: How long to wait when sending a command
//   - MaxRetries:   How many times to retry on transient failures
type Config struct {
	// ==========================================================================
	// REQUIRED FIELDS
	// ==========================================================================

	// Addr is the Redis server address in "host:port" format.
	// Examples:
	//   - "localhost:6379"         -> Redis running on your machine
	//   - "redis.example.com:6379" -> Redis running on a remote server
	//
	// VALIDATION: Cannot be empty.
	Addr string

	// Password for Redis authentication.
	// Leave empty ("") if your Redis server doesn't require a password.
	// In production, you should ALWAYS set a password!
	Password string

	// DB is the database number to use (0-15).
	// Redis has 16 separate databases - they're like separate namespaces.
	// Most apps just use DB 0 (the default).
	// You might use different DBs for different environments (dev vs test).
	//
	// VALIDATION: Must be between 0 and 15 (inclusive).
	DB int

	// ==========================================================================
	// OPTIONAL FIELDS (Connection Pool & Timeouts)
	// ==========================================================================

	// PoolSize is the maximum number of connections to keep open.
	//
	// WHY USE A POOL?
	// Opening a new connection for every command is slow (~1ms overhead).
	// A pool keeps connections open and ready, so commands are faster.
	//
	// CHOOSING A SIZE:
	//   - Too small: Commands wait for a free connection (higher latency)
	//   - Too large: Wasted memory and connections
	//   - Default (0): Uses 10 * runtime.GOMAXPROCS connections
	//
	// For most apps, the default is fine. Increase if you see "connection pool exhausted" errors.
	PoolSize int

	// MinIdleConns is the minimum number of idle connections to keep open.
	//
	// WHY KEEP IDLE CONNECTIONS?
	// Cold connections take time to establish. Keeping some warm means
	// the first few commands don't have extra latency.
	//
	// Default (0): No minimum - connections are created on demand.
	MinIdleConns int

	// DialTimeout is how long to wait when establishing a new connection.
	//
	// If Redis is slow to respond (network issues, overloaded), this prevents
	// your app from hanging forever.
	//
	// Default (0): 5 seconds
	DialTimeout time.Duration

	// ReadTimeout is how long to wait for Redis to respond to a command.
	//
	// If a command takes longer than this, it fails with a timeout error.
	// This protects against Redis being overloaded or stuck.
	//
	// Default (0): 3 seconds
	// Set to -1 to disable (not recommended in production)
	ReadTimeout time.Duration

	// WriteTimeout is how long to wait when sending a command to Redis.
	//
	// If the network is congested, this prevents your app from blocking forever.
	//
	// Default (0): Same as ReadTimeout
	WriteTimeout time.Duration

	// MaxRetries is how many times to retry a command on transient failures.
	//
	// WHAT'S A TRANSIENT FAILURE?
	// Temporary issues like network blips or Redis being briefly unavailable.
	// Retrying often succeeds on the second or third attempt.
	//
	// Default (0): No retries (fail immediately)
	// Recommended: 3 for production
	MaxRetries int
}

// Validate checks that the configuration is valid.
//
// This is called automatically by NewClient, but you can call it yourself
// if you want to validate config before attempting to connect.
//
// VALIDATION RULES:
//   - Addr cannot be empty
//   - DB must be between 0 and 15
func (c Config) Validate() error {
	if c.Addr == "" {
		return ErrEmptyAddr
	}
	if c.DB < 0 || c.DB > 15 {
		return ErrInvalidDB
	}
	return nil
}

// =============================================================================
// CLIENT CREATION
// =============================================================================

// NewClient creates a new Redis client and verifies the connection works.
//
// HOW IT WORKS:
//  1. Validates the configuration (returns error if invalid)
//  2. Creates a new Redis client with your config settings
//  3. Sends a "PING" command to make sure Redis is actually reachable
//  4. Returns the client if successful, or an error if something went wrong
//
// EXAMPLE USAGE:
//
//	client, err := redis.NewClient(redis.Config{
//	    Addr:     "localhost:6379",
//	    Password: "",  // no password for local development
//	    DB:       0,   // use default database
//	})
//	if err != nil {
//	    log.Fatal("Could not connect to Redis:", err)
//	}
//	defer client.Close()  // Don't forget to close when done!
//
// PRODUCTION EXAMPLE:
//
//	client, err := redis.NewClient(redis.Config{
//	    Addr:         "redis.prod.example.com:6379",
//	    Password:     os.Getenv("REDIS_PASSWORD"),
//	    DB:           0,
//	    PoolSize:     100,
//	    MinIdleConns: 10,
//	    DialTimeout:  5 * time.Second,
//	    ReadTimeout:  3 * time.Second,
//	    WriteTimeout: 3 * time.Second,
//	    MaxRetries:   3,
//	})
func NewClient(cfg Config) (*redis.Client, error) {
	// First, validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	// Create the Redis client with our configuration.
	// This doesn't actually connect yet - it just sets up the client.
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,

		// Connection pool settings
		PoolSize:     cfg.PoolSize,     // 0 = default (10 * GOMAXPROCS)
		MinIdleConns: cfg.MinIdleConns, // 0 = no minimum

		// Timeout settings
		DialTimeout:  cfg.DialTimeout,  // 0 = default (5s)
		ReadTimeout:  cfg.ReadTimeout,  // 0 = default (3s)
		WriteTimeout: cfg.WriteTimeout, // 0 = default (same as read)

		// Retry settings
		MaxRetries: cfg.MaxRetries, // 0 = no retries
	})

	// Now let's actually try to connect by sending a PING command.
	// This is important because we want to fail fast if Redis isn't available,
	// rather than getting weird errors later when we try to use it.
	if err := Ping(context.Background(), client); err != nil {
		// Close the client to clean up any resources
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// Ping sends a PING command to Redis and checks if it responds with PONG.
//
// WHY IS THIS USEFUL?
// It's the simplest way to check if:
//   - Redis is running
//   - We can reach it over the network
//   - Our authentication (password) is correct
//
// The 'ctx' parameter allows you to set a timeout or cancel the operation.
// For example, you might want to give up after 5 seconds if Redis doesn't respond.
//
// EXAMPLE - Health check endpoint:
//
//	func healthHandler(w http.ResponseWriter, r *http.Request) {
//	    ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
//	    defer cancel()
//
//	    if err := redis.Ping(ctx, redisClient); err != nil {
//	        http.Error(w, "Redis unhealthy", http.StatusServiceUnavailable)
//	        return
//	    }
//	    w.WriteHeader(http.StatusOK)
//	}
func Ping(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return errors.New("redis: client is nil")
	}

	// client.Ping() sends the PING command to Redis.
	// Redis should respond with "PONG" if everything is working.
	// .Err() gives us any error that occurred (nil if successful).
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// =============================================================================
// GRACEFUL SHUTDOWN
// =============================================================================

// Close closes the Redis client connection.
//
// This is just a convenience wrapper around client.Close() that handles nil.
// You should call this when your application is shutting down.
//
// EXAMPLE:
//
//	client, err := redis.NewClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer redis.Close(client)  // Safe even if client is nil
func Close(client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
