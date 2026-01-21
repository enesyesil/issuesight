package log

import (
	"context"
	"log/slog"
)

// Context keys for storing values.
//
// WHY CUSTOM TYPE?
// Go recommends using a custom type for context keys to avoid collisions.
// If two packages both use string("logger") as a key, they'd conflict.
// Using our own type `ctxKey` ensures only this package can access these values.
type ctxKey string

const (
	loggerKey    ctxKey = "logger"
	requestIDKey ctxKey = "request_id"
)

// WithLogger stores a logger in the context.
//
// WHY STORE LOGGER IN CONTEXT?
// Allows passing a configured logger through the call chain without
// adding it as a parameter to every function.
//
// COMMON PATTERN:
//
//	func handleRequest(ctx context.Context, req *Request) {
//	    // Create logger with request-specific fields
//	    logger := log.Default().With("path", req.Path, "method", req.Method)
//	    ctx = log.WithLogger(ctx, logger)
//
//	    // All downstream functions can get this logger
//	    processRequest(ctx, req)
//	}
//
//	func processRequest(ctx context.Context, req *Request) {
//	    log.FromContext(ctx).Info("processing")  // Has path and method fields
//	}
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	// Handle nil context gracefully - create a new background context
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context.
// Returns the default logger if none is stored.
//
// WHY RETURN DEFAULT?
// Ensures you always get a working logger, even if context is nil
// or doesn't have a logger stored. No need for nil checks.
//
// USAGE:
//
//	func doSomething(ctx context.Context) {
//	    logger := log.FromContext(ctx)  // Always returns a valid logger
//	    logger.Info("doing something")
//	}
func FromContext(ctx context.Context) *slog.Logger {
	// Handle nil context - return default logger
	if ctx == nil {
		return defaultLogger
	}

	// Try to get logger from context
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
		return logger
	}

	// Fall back to default logger
	return defaultLogger
}

// WithRequestID stores a request ID in the context.
//
// WHAT IS A REQUEST ID?
// A unique identifier for a single request/operation. Usually a UUID.
// Allows tracing all logs from one request across services.
//
// COMMON PATTERN (in HTTP middleware):
//
//	func RequestIDMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        requestID := uuid.NewString()
//	        ctx := log.WithRequestID(r.Context(), requestID)
//	        next.ServeHTTP(w, r.WithContext(ctx))
//	    })
//	}
//
// Now all logs in that request include: request_id=abc-123-def
func WithRequestID(ctx context.Context, id string) context.Context {
	// Handle nil context gracefully
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext retrieves the request ID from context.
// Returns empty string if not found.
//
// USAGE:
// Usually called by the logging system itself (see wrappedHandler.Handle).
// You rarely need to call this directly.
func RequestIDFromContext(ctx context.Context) string {
	// Handle nil context
	if ctx == nil {
		return ""
	}

	// Try to get request ID from context
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}

	// Not found
	return ""
}

// WithFields returns a logger with additional fields from context.
// Useful for adding request-scoped data to all logs.
//
// WHAT THIS DOES:
//  1. Gets the logger from context (or default)
//  2. Adds the request ID if present
//  3. Adds any extra fields you pass
//  4. Returns the enriched logger
//
// USAGE:
//
//	func handleOrder(ctx context.Context, orderID string) {
//	    // Create a logger with order-specific fields
//	    logger := log.WithFields(ctx, "order_id", orderID)
//
//	    logger.Info("processing order")     // Has request_id + order_id
//	    logger.Info("validating payment")   // Has request_id + order_id
//	}
//
// WHY NOT USE logger.With() DIRECTLY?
// This helper also adds the request ID from context automatically.
func WithFields(ctx context.Context, args ...any) *slog.Logger {
	// Get logger from context (or default)
	logger := FromContext(ctx)

	// Add request ID if present in context
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		logger = logger.With("request_id", reqID)
	}

	// Add any extra fields passed by caller
	if len(args) > 0 {
		logger = logger.With(args...)
	}

	return logger
}
