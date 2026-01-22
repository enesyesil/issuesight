package log

import (
	"context"
	"io"
	"log/slog"
)

// newHandler creates the appropriate handler based on config.
//
// WHAT IS A HANDLER?
// In slog, a Handler decides HOW logs are formatted and WHERE they go.
// slog provides two built-in handlers:
//   - JSONHandler: outputs {"time":"...", "level":"INFO", "msg":"...", "key":"value"}
//   - TextHandler: outputs time=... level=INFO msg=... key=value
//
// HOW IT WORKS:
//  1. Creates HandlerOptions with the log level and attribute replacer
//  2. Based on format, returns either JSON or text handler
//  3. The ReplaceAttr function is called for EVERY attribute, allowing redaction
func newHandler(cfg Config, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		// Level: only log messages at this level or higher
		// If level is WARN, DEBUG and INFO messages are ignored
		Level: level,

		// ReplaceAttr: called for each attribute before it's written
		// We use this to redact sensitive values like passwords and tokens
		ReplaceAttr: replaceAttr,
	}

	// Choose handler based on format
	switch cfg.Format {
	case "json":
		// JSONHandler outputs one JSON object per line
		// Perfect for log aggregators (Datadog, Splunk, ELK, etc.)
		return slog.NewJSONHandler(cfg.Output, opts)
	default:
		// TextHandler outputs human-readable key=value format
		// Easier to read during development
		return slog.NewTextHandler(cfg.Output, opts)
	}
}

// replaceAttr modifies log attributes before output.
// Used for redaction and formatting.
//
// WHEN IS THIS CALLED?
// slog calls this function for EVERY attribute in EVERY log message.
// It receives the attribute and can return a modified version.
//
// WHAT WE DO:
// Check if the key looks sensitive (contains "password", "token", etc.)
// If so, replace the value with "[REDACTED]"
//
// WHY?
// Prevents accidentally logging secrets:
//
//	log.Info("connected", "token", apiToken)
//	// Output: level=INFO msg=connected token=[REDACTED]
//
// PARAMETERS:
//   - groups: nested group names (for grouped attributes, rarely used)
//   - a: the attribute (key-value pair) being logged
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	// Redact sensitive values
	if shouldRedact(a.Key) {
		return slog.String(a.Key, "[REDACTED]")
	}
	return a
}

// wrappedHandler wraps a handler to inject context values.
//
// WHY WRAP THE HANDLER?
// slog's built-in handlers don't automatically extract values from context.
// This wrapper intercepts every log call and adds the request ID if present.
//
// DECORATOR PATTERN:
// wrappedHandler "decorates" another handler, adding behavior before delegating.
// It implements slog.Handler interface so it can be used like any handler.
type wrappedHandler struct {
	handler slog.Handler // The actual handler (JSON or text) we delegate to
}

// Enabled returns whether the handler is enabled for the given level.
//
// WHAT THIS DOES:
// Checks if a message at this level should be logged.
// Delegates to the underlying handler (which checks against configured level).
//
// WHY NEEDED:
// slog calls Enabled() before Handle() as an optimization.
// If Enabled() returns false, Handle() is never called.
func (h *wrappedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle processes a log record.
//
// WHAT THIS DOES:
//  1. Checks if context has a request ID
//  2. If so, adds it as an attribute to the record
//  3. Delegates to the underlying handler to actually write the log
//
// WHY USEFUL:
// Request ID lets you trace all logs from a single HTTP request.
// Set it once at the start of the request, and it appears in all logs.
//
//	ctx = log.WithRequestID(ctx, "req-abc123")
//	log.FromContext(ctx).Info("step 1")  // includes request_id=req-abc123
//	log.FromContext(ctx).Info("step 2")  // includes request_id=req-abc123
func (h *wrappedHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add request ID from context if present
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	return h.handler.Handle(ctx, r)
}

// WithAttrs returns a new handler with additional attributes.
//
// WHAT THIS DOES:
// Creates a new handler that includes the given attributes in every log.
// Called when you do: logger.With("key", "value")
//
// WHY WRAP IT:
// We need to maintain the wrapping, so context extraction still works.
func (h *wrappedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &wrappedHandler{handler: h.handler.WithAttrs(attrs)}
}

// WithGroup returns a new handler with a group name.
//
// WHAT THIS DOES:
// Creates a new handler that nests subsequent attributes under a group.
// Called when you do: logger.WithGroup("http")
//
// Example:
//
//	logger.WithGroup("http").Info("request", "method", "GET", "path", "/api")
//	// Output: http.method=GET http.path=/api
func (h *wrappedHandler) WithGroup(name string) slog.Handler {
	return &wrappedHandler{handler: h.handler.WithGroup(name)}
}

// NewWithContext creates a logger that automatically extracts context values.
//
// DIFFERENCE FROM New():
// New() creates a standard logger.
// NewWithContext() creates a logger that automatically adds request_id
// from context on every log call.
//
// USE WHEN:
// You want request IDs added automatically without calling WithFields().
func NewWithContext(cfg Config) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = io.Discard
	}

	level := ParseLevel(cfg.Level)

	// Create the base handler (JSON or text)
	baseHandler := newHandler(cfg, level)

	// Wrap it to inject context values
	wrapped := &wrappedHandler{handler: baseHandler}

	// Create logger with wrapped handler
	logger := slog.New(wrapped)

	// Add service name if provided
	if cfg.Service != "" {
		logger = logger.With("service", cfg.Service)
	}

	return logger
}
