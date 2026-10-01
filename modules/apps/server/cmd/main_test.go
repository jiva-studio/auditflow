package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	_ = os.Unsetenv("PORT")
	cfg := LoadConfig()
	if cfg.Port != ":8080" {
		t.Fatalf("expected default port :8080, got %s", cfg.Port)
	}
}

func TestLoadConfig_Custom(t *testing.T) {
	_ = os.Setenv("PORT", "9090")
	defer func() { _ = os.Unsetenv("PORT") }()

	cfg := LoadConfig()
	if cfg.Port != ":9090" {
		t.Fatalf("expected custom port :9090, got %s", cfg.Port)
	}
}

func TestRun_Lifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{Port: ":0"} // dynamic free port

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, cfg)
	}()

	time.Sleep(100 * time.Millisecond)

	// Cancel context to initiate graceful shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("unexpected error during Run lifecycle: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to shut down")
	}
}
