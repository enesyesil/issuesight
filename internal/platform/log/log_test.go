package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"warn", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},           // empty defaults to INFO
		{"invalid", slog.LevelInfo},    // invalid defaults to INFO
		{"  INFO  ", slog.LevelInfo},   // whitespace trimmed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLevel(tt.input)
			if got != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLevelToString(t *testing.T) {
	tests := []struct {
		input    slog.Level
		expected string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := LevelToString(tt.input)
			if got != tt.expected {
				t.Errorf("LevelToString(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	var buf bytes.Buffer

	logger := New(Config{
		Level:   "DEBUG",
		Format:  "text",
		Service: "test-service",
		Output:  &buf,
	})

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got %q", output)
	}
	if !strings.Contains(output, "service=test-service") {
		t.Errorf("expected output to contain service name, got %q", output)
	}
}

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer

	logger := New(Config{
		Level:  "INFO",
		Format: "json",
		Output: &buf,
	})

	logger.Info("json test")

	output := buf.String()
	if !strings.Contains(output, `"msg":"json test"`) {
		t.Errorf("expected JSON output, got %q", output)
	}
}

func TestNew_NilOutput(t *testing.T) {
	// Should not panic with nil output
	logger := New(Config{
		Level:  "INFO",
		Format: "text",
		Output: nil,
	})

	// Should use os.Stdout, just verify no panic
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestShouldRedact(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"PASSWORD", true},
		{"api_key", true},
		{"API_KEY", true},
		{"token", true},
		{"access_token", true},
		{"secret", true},
		{"authorization", true},
		{"auth_header", true},
		{"credential", true},
		{"username", false},
		{"email", false},
		{"name", false},
		{"id", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := shouldRedact(tt.key)
			if got != tt.expected {
				t.Errorf("shouldRedact(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestRedact(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"password", "secret123", "[REDACTED]"},
		{"api_key", "sk-xxx", "[REDACTED]"},
		{"username", "john", "john"},
		{"email", "john@example.com", "john@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := Redact(tt.key, tt.value)
			if got != tt.expected {
				t.Errorf("Redact(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.expected)
			}
		})
	}
}

func TestRedactMap(t *testing.T) {
	input := map[string]string{
		"username": "john",
		"password": "secret123",
		"api_key":  "sk-xxx",
	}

	result := RedactMap(input)

	if result["username"] != "john" {
		t.Errorf("username should not be redacted")
	}
	if result["password"] != "[REDACTED]" {
		t.Errorf("password should be redacted")
	}
	if result["api_key"] != "[REDACTED]" {
		t.Errorf("api_key should be redacted")
	}
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  "INFO",
		Format: "text",
		Output: &buf,
	})

	ctx := context.Background()
	ctx = WithLogger(ctx, logger)

	retrieved := FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected non-nil logger from context")
	}

	retrieved.Info("context test")
	if !strings.Contains(buf.String(), "context test") {
		t.Error("expected log output from context logger")
	}
}

func TestFromContext_NilContext(t *testing.T) {
	// Should return default logger, not panic
	logger := FromContext(nil)
	if logger == nil {
		t.Error("expected non-nil default logger")
	}
}

func TestFromContext_NoLogger(t *testing.T) {
	// Should return default logger when none stored
	ctx := context.Background()
	logger := FromContext(ctx)
	if logger == nil {
		t.Error("expected non-nil default logger")
	}
}

func TestRequestID(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")

	got := RequestIDFromContext(ctx)
	if got != "req-123" {
		t.Errorf("RequestIDFromContext() = %q, want %q", got, "req-123")
	}
}

func TestRequestIDFromContext_NilContext(t *testing.T) {
	got := RequestIDFromContext(nil)
	if got != "" {
		t.Errorf("expected empty string for nil context, got %q", got)
	}
}

func TestWithRequestID_NilContext(t *testing.T) {
	// Should not panic
	ctx := WithRequestID(nil, "req-123")
	if ctx == nil {
		t.Error("expected non-nil context")
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  "INFO",
		Format: "text",
		Output: &buf,
	})

	ctx := context.Background()
	ctx = WithLogger(ctx, logger)
	ctx = WithRequestID(ctx, "req-456")

	l := WithFields(ctx, "extra", "field")
	l.Info("with fields test")

	output := buf.String()
	if !strings.Contains(output, "request_id=req-456") {
		t.Errorf("expected request_id in output, got %q", output)
	}
	if !strings.Contains(output, "extra=field") {
		t.Errorf("expected extra field in output, got %q", output)
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  "DEBUG",
		Format: "text",
		Output: &buf,
	})
	SetDefault(logger)

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	output := buf.String()
	if !strings.Contains(output, "debug msg") {
		t.Error("expected debug message")
	}
	if !strings.Contains(output, "info msg") {
		t.Error("expected info message")
	}
	if !strings.Contains(output, "warn msg") {
		t.Error("expected warn message")
	}
	if !strings.Contains(output, "error msg") {
		t.Error("expected error message")
	}
}
