// Package log provides structured logging using slog.
//
// WHY THIS PACKAGE EXISTS:
// Go 1.21+ includes "log/slog" - a structured logging library in the stdlib.
// This package wraps slog to provide:
//   - Easy configuration via Config struct
//   - Automatic service name injection
//   - Sensitive data redaction
//   - Context-based request ID propagation
//   - Package-level convenience functions
//
// WHAT IS STRUCTURED LOGGING?
// Instead of: log.Println("Failed to fetch issue 123 from kubernetes/kubernetes")
// We write:   log.Error("fetch failed", "issue", 123, "owner", "kubernetes", "repo", "kubernetes")
//
// Benefits:
//   - Machine-parseable (JSON output)
//   - Searchable in log aggregators (Datadog, Splunk, etc.)
//   - Consistent format across all services
//
// USAGE:
//
//	logger := log.New(log.Config{
//	    Level:   "INFO",
//	    Format:  "json",
//	    Service: "collector",
//	})
//	log.SetDefault(logger)
//	log.Info("server started", "port", 8080)
package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config holds logging configuration.
//
// WHY A CONFIG STRUCT?
// Makes it easy to load settings from environment variables or config files.
// All services use the same Config structure for consistency.
type Config struct {
	// Level controls which messages are logged.
	// DEBUG < INFO < WARN < ERROR
	// If Level is "WARN", only WARN and ERROR messages are logged.
	Level string // DEBUG, INFO, WARN, ERROR

	// Format controls the output format.
	// "json" - Machine-readable, for production (parsed by log aggregators)
	// "text" - Human-readable, for development (easier to read in terminal)
	Format string // json, text

	// Service is the name of the service, added to every log entry.
	// Helps filter logs when multiple services write to the same destination.
	Service string // service name added to all logs

	// Output is where logs are written. Usually os.Stdout.
	// Can be set to a file, buffer (for testing), or io.Discard.
	Output io.Writer
}

// DefaultConfig returns sensible defaults for development.
//
// WHY DEFAULTS?
// So you can quickly create a logger without specifying everything:
//
//	logger := log.New(log.DefaultConfig())
func DefaultConfig() Config {
	return Config{
		Level:  "INFO",
		Format: "text",
		Output: os.Stdout,
	}
}

// New creates a configured slog.Logger.
//
// HOW IT WORKS:
//  1. Validates/defaults the output writer
//  2. Parses the level string to slog.Level
//  3. Creates the appropriate handler (JSON or text)
//  4. Wraps it in a slog.Logger
//  5. Adds the service name as a permanent field (if provided)
//
// The returned logger is thread-safe and can be shared across goroutines.
func New(cfg Config) *slog.Logger {
	// Default to stdout if no output specified
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	// Convert string level to slog.Level
	level := ParseLevel(cfg.Level)

	// Create handler (JSON or text) with redaction support
	handler := newHandler(cfg, level)

	// Create the logger with our handler
	logger := slog.New(handler)

	// Add service name if provided - this field appears in every log entry
	// Example: {"service": "collector", "msg": "started", ...}
	if cfg.Service != "" {
		logger = logger.With("service", cfg.Service)
	}

	return logger
}

// ParseLevel converts a string to slog.Level.
// Returns INFO for invalid/empty strings.
//
// WHY STRING INPUT?
// Environment variables are strings. This lets you do:
//
//	LOG_LEVEL=DEBUG ./myservice
//
// And the service reads it via os.Getenv("LOG_LEVEL").
//
// CASE INSENSITIVE:
// "debug", "DEBUG", "Debug" all work the same.
func ParseLevel(s string) slog.Level {
	// Normalize: trim whitespace and convert to uppercase
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug // Most verbose, for debugging
	case "INFO":
		return slog.LevelInfo // Normal operations
	case "WARN", "WARNING":
		return slog.LevelWarn // Something unexpected but handled
	case "ERROR":
		return slog.LevelError // Something failed
	default:
		return slog.LevelInfo // Safe default if invalid
	}
}

// LevelToString converts slog.Level to string.
//
// WHY THIS?
// Useful for logging the current level, or for config validation messages.
func LevelToString(l slog.Level) string {
	switch l {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Default logger for package-level functions.
//
// WHY A DEFAULT LOGGER?
// Convenience. Instead of passing a logger everywhere, you can just use:
//
//	log.Info("something happened")
//
// The default is initialized with DefaultConfig() on package load,
// but you should call SetDefault() with your configured logger at startup.
var defaultLogger = New(DefaultConfig())

// SetDefault sets the default logger.
//
// CALL THIS AT STARTUP:
//
//	func main() {
//	    logger := log.New(log.Config{...})
//	    log.SetDefault(logger)
//	    // Now log.Info(), log.Error() etc. use your configured logger
//	}
//
// Also sets slog.SetDefault so third-party libraries using slog directly
// will use the same logger.
func SetDefault(l *slog.Logger) {
	defaultLogger = l
	slog.SetDefault(l) // Also set stdlib default
}

// Default returns the default logger.
//
// USE WHEN:
// You need to pass the logger explicitly (e.g., to a struct field).
func Default() *slog.Logger {
	return defaultLogger
}

// Package-level logging functions using default logger.
//
// WHY THESE FUNCTIONS?
// Convenience. Instead of:
//
//	log.Default().Info("message", "key", value)
//
// You can write:
//
//	log.Info("message", "key", value)
//
// ARGS FORMAT:
// Arguments are key-value pairs: "key1", value1, "key2", value2, ...
// Example: log.Info("user created", "id", 123, "email", "john@example.com")

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}
