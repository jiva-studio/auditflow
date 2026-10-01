package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestGenerateTraceID(t *testing.T) {
	id1 := GenerateTraceID()
	id2 := GenerateTraceID()

	if len(id1) != 32 {
		t.Fatalf("expected 32-character hex trace ID, got length %d (%s)", len(id1), id1)
	}
	if id1 == id2 {
		t.Fatalf("expected unique trace IDs, got %s and %s", id1, id2)
	}
}

func TestWithTraceID_And_FromContext(t *testing.T) {
	ctx := context.Background()
	if id := TraceIDFromContext(ctx); id != "" {
		t.Fatalf("expected empty trace ID from empty context, got %s", id)
	}

	var nilCtx context.Context
	if id := TraceIDFromContext(nilCtx); id != "" {
		t.Fatalf("expected empty trace ID from nil context, got %s", id)
	}

	traceID := "abcd1234efgh5678"
	ctxWithID := WithTraceID(ctx, traceID)
	if got := TraceIDFromContext(ctxWithID); got != traceID {
		t.Fatalf("expected trace ID %s, got %s", traceID, got)
	}

	nilCtxWithID := WithTraceID(nilCtx, traceID)
	if got := TraceIDFromContext(nilCtxWithID); got != traceID {
		t.Fatalf("expected trace ID %s from nil-initialized context, got %s", traceID, got)
	}
}

func TestWithLogger_And_FromContext(t *testing.T) {
	var buf bytes.Buffer
	customLogger := slog.New(slog.NewTextHandler(&buf, nil))

	ctx := context.Background()
	if got := LoggerFromContext(ctx); got != slog.Default() {
		t.Fatalf("expected default logger for unconfigured context")
	}

	var nilCtx context.Context
	if got := LoggerFromContext(nilCtx); got != slog.Default() {
		t.Fatalf("expected default logger for nil context")
	}

	ctxWithLogger := WithLogger(ctx, customLogger)
	if got := LoggerFromContext(ctxWithLogger); got != customLogger {
		t.Fatalf("expected custom logger from context")
	}

	ctxWithNil := WithLogger(ctx, nil)
	if got := LoggerFromContext(ctxWithNil); got == nil {
		t.Fatalf("expected fallback logger from WithLogger(ctx, nil)")
	}
}
