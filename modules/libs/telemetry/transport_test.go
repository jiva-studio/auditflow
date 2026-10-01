package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTraceRoundTripper_PropagatesContextTraceID(t *testing.T) {
	expectedTrace := "trace-12345678-abcd"
	var receivedTrace string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTrace = r.Header.Get(HeaderTraceID)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: NewTraceRoundTripper(http.DefaultTransport),
	}

	ctx := WithTraceID(context.Background(), expectedTrace)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if receivedTrace != expectedTrace {
		t.Fatalf("expected trace %s at downstream server, got %s", expectedTrace, receivedTrace)
	}
}

func TestTraceRoundTripper_GeneratesTraceIDIfMissing(t *testing.T) {
	var receivedTrace string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTrace = r.Header.Get(HeaderTraceID)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: NewTraceRoundTripper(nil),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if len(receivedTrace) != 32 {
		t.Fatalf("expected generated 32-char hex trace ID, got: %s", receivedTrace)
	}
}
