package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTraceMiddleware_GeneratesTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{
		ServiceName: "test-service",
		Format:      FormatJSON,
		Output:      &buf,
	})

	handler := TraceMiddleware("test-service", logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxTrace := TraceIDFromContext(r.Context())
		if ctxTrace == "" {
			t.Errorf("expected trace ID in context")
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	respTrace := rec.Header().Get(HeaderTraceID)
	if respTrace == "" {
		t.Fatalf("expected X-Trace-ID header in response")
	}

	if len(respTrace) != 32 {
		t.Errorf("expected 32-char hex trace ID, got: %s", respTrace)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v, raw: %s", err, buf.String())
	}

	if logEntry["trace_id"] != respTrace {
		t.Errorf("expected log trace_id %s, got %v", respTrace, logEntry["trace_id"])
	}
	if logEntry["method"] != "POST" {
		t.Errorf("expected POST method, got %v", logEntry["method"])
	}
	if logEntry["status"] != float64(http.StatusAccepted) {
		t.Errorf("expected status 202, got %v", logEntry["status"])
	}
}

func TestTraceMiddleware_PreservesIncomingTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{
		ServiceName: "test-service",
		Format:      FormatJSON,
		Output:      &buf,
	})

	incomingTraceID := "custom-trace-id-12345"

	handler := TraceMiddleware("test-service", logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := TraceIDFromContext(r.Context()); got != incomingTraceID {
			t.Errorf("expected context trace ID %s, got %s", incomingTraceID, got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/tick", nil)
	req.Header.Set(HeaderTraceID, incomingTraceID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderTraceID); got != incomingTraceID {
		t.Fatalf("expected response header %s, got %s", incomingTraceID, got)
	}
}
