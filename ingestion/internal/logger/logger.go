// Package logger provides structured JSON logging with correlation ID
// propagation and session identifier redaction for the Kasumi Engine
// ingestion service.
//
// The log schema is consistent across Go and Python services:
//
//	{"timestamp": "...", "level": "...", "service": "...", "correlation_id": "...", "message": "..."}
package logger

import (
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const (
	// correlationIDKey is the context key for the correlation ID.
	correlationIDKey contextKey = "correlation_id"
)

// sessionIDPattern matches common session ID formats for redaction.
// Matches UUIDs, hex strings of 16+ chars, and base64-like tokens.
var sessionIDPattern = regexp.MustCompile(
	`(?i)(session[_-]?id|sid|sess)["\s:=]+["']?([a-zA-Z0-9+/=_-]{16,})["']?`,
)

// redactionReplacement is the string used to replace redacted session IDs.
const redactionReplacement = "[REDACTED]"

// redactingWriter wraps an io.Writer and redacts session identifiers.
type redactingWriter struct {
	w io.Writer
}

func (rw *redactingWriter) Write(p []byte) (n int, err error) {
	redacted := sessionIDPattern.ReplaceAll(p, []byte("$1: "+redactionReplacement))
	return rw.w.Write(redacted)
}

// New creates a new zerolog.Logger configured for the Kasumi Engine.
//
// Parameters:
//   - level: log level string ("debug", "info", "warn", "error")
//   - service: service name for the "service" field
//
// The logger outputs JSON with a consistent schema matching the Python
// structlog configuration.
func New(level string, service string) zerolog.Logger {
	return NewWithOptions(level, service, true, os.Stdout)
}

// NewWithOptions creates a new zerolog.Logger with explicit options.
//
// Parameters:
//   - level: log level string
//   - service: service name
//   - redact: whether to redact session identifiers from log output
//   - output: the io.Writer to write logs to
func NewWithOptions(level string, service string, redact bool, output io.Writer) zerolog.Logger {
	// Set up zerolog globals
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFieldName = "level"
	zerolog.MessageFieldName = "message"

	// Parse log level
	zLevel := parseLevel(level)

	// Optionally wrap the writer with redaction
	var writer io.Writer = output
	if redact {
		writer = &redactingWriter{w: output}
	}

	// Build the logger
	logger := zerolog.New(writer).
		Level(zLevel).
		With().
		Timestamp().
		Str("service", service).
		Logger()

	return logger
}

// WithCorrelationID adds a correlation ID to the context for propagation.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// CorrelationIDFromContext extracts the correlation ID from the context.
// Returns an empty string if not present.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithContext returns a logger with the correlation ID from the context.
func WithContext(log zerolog.Logger, ctx context.Context) zerolog.Logger {
	correlationID := CorrelationIDFromContext(ctx)
	if correlationID != "" {
		return log.With().Str("correlation_id", correlationID).Logger()
	}
	return log
}

// parseLevel converts a string log level to a zerolog.Level.
func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
