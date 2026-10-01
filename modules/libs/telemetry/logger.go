package telemetry

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// LogFormat defines the serialization format for slog records.
type LogFormat string

const (
	// FormatJSON serializes logs as JSON objects (recommended for production).
	FormatJSON LogFormat = "json"
	// FormatText serializes logs as human-readable key-value text.
	FormatText LogFormat = "text"
)

// LoggerConfig holds configuration options for building a slog.Logger.
type LoggerConfig struct {
	ServiceName string
	Level       slog.Level
	Format      LogFormat
	Output      io.Writer
}

// NewLogger instantiates a structured slog.Logger based on the provided configuration.
func NewLogger(cfg LoggerConfig) *slog.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level: cfg.Level,
	}

	var handler slog.Handler
	switch LogFormat(strings.ToLower(string(cfg.Format))) {
	case FormatText:
		handler = slog.NewTextHandler(out, opts)
	case FormatJSON:
		fallthrough
	default:
		handler = slog.NewJSONHandler(out, opts)
	}

	logger := slog.New(handler)
	if strings.TrimSpace(cfg.ServiceName) != "" {
		logger = logger.With(slog.String("service", strings.TrimSpace(cfg.ServiceName)))
	}

	return logger
}
