package telemetry

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLogger_JSON(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{
		ServiceName: "test-service",
		Level:       slog.LevelInfo,
		Format:      FormatJSON,
		Output:      &buf,
	})

	logger.Info("sample event", slog.String("key", "val"))

	out := buf.String()
	var logMap map[string]interface{}
	if err := json.Unmarshal([]byte(out), &logMap); err != nil {
		t.Fatalf("expected valid JSON log output, got error: %v, raw: %s", err, out)
	}

	if logMap["msg"] != "sample event" {
		t.Errorf("expected msg 'sample event', got %v", logMap["msg"])
	}
	if logMap["service"] != "test-service" {
		t.Errorf("expected service 'test-service', got %v", logMap["service"])
	}
	if logMap["key"] != "val" {
		t.Errorf("expected key 'val', got %v", logMap["key"])
	}
}

func TestNewLogger_Text(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{
		ServiceName: "agent",
		Level:       slog.LevelDebug,
		Format:      FormatText,
		Output:      &buf,
	})

	logger.Debug("debug message", slog.Int("count", 42))

	out := buf.String()
	if !strings.Contains(out, "service=agent") {
		t.Errorf("expected output to contain 'service=agent', got: %s", out)
	}
	if !strings.Contains(out, "count=42") {
		t.Errorf("expected output to contain 'count=42', got: %s", out)
	}
}
