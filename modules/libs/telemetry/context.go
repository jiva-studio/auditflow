// Package telemetry provides structured logging and distributed tracing utilities using log/slog.
package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

type contextKey string

const (
	traceIDKey contextKey = "telemetry.trace_id"
	loggerKey  contextKey = "telemetry.logger"

	// HeaderTraceID is the canonical HTTP header used to propagate trace identifiers across services.
	HeaderTraceID = "X-Trace-ID"
)

// GenerateTraceID creates a new cryptographically random 16-byte hex-encoded trace ID.
func GenerateTraceID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}

// WithTraceID returns a new context containing the given trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext extracts the trace ID from the context, or returns an empty string.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

// WithLogger returns a new context containing the given logger.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext retrieves the logger from the context, falling back to slog.Default().
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
			return logger
		}
	}
	return slog.Default()
}
