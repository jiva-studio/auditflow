package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnv(t *testing.T) {
	key := "STREAMER_TEST_ENV_VAR"
	_ = os.Unsetenv(key)

	if got := getEnv(key, "default_val"); got != "default_val" {
		t.Errorf("got %q, want default_val", got)
	}

	_ = os.Setenv(key, "custom_val")
	defer func() { _ = os.Unsetenv(key) }()

	if got := getEnv(key, "default_val"); got != "custom_val" {
		t.Errorf("got %q, want custom_val", got)
	}
}

func TestRun_ConfigValidation(t *testing.T) {
	t.Run("invalid tick_ms", func(t *testing.T) {
		_ = os.Setenv("TICK_MS", "invalid")
		defer func() { _ = os.Unsetenv("TICK_MS") }()

		err := run()
		if err == nil {
			t.Fatalf("expected error for invalid TICK_MS, got nil")
		}
	})

	t.Run("negative tick_ms", func(t *testing.T) {
		_ = os.Setenv("TICK_MS", "-10")
		defer func() { _ = os.Unsetenv("TICK_MS") }()

		err := run()
		if err == nil {
			t.Fatalf("expected error for negative TICK_MS, got nil")
		}
	})

	t.Run("empty agent_url", func(t *testing.T) {
		_ = os.Setenv("TICK_MS", "100")
		_ = os.Setenv("AGENT_URL", "   ")
		defer func() {
			_ = os.Unsetenv("TICK_MS")
			_ = os.Unsetenv("AGENT_URL")
		}()

		err := run()
		if err == nil {
			t.Fatalf("expected error for empty AGENT_URL, got nil")
		}
	})

	t.Run("recording not found", func(t *testing.T) {
		_ = os.Setenv("TICK_MS", "100")
		_ = os.Setenv("AGENT_URL", "http://localhost:8081")
		_ = os.Setenv("RECORDING_PATH", "/non/existent/recording.tar.gz")
		defer func() {
			_ = os.Unsetenv("TICK_MS")
			_ = os.Unsetenv("AGENT_URL")
			_ = os.Unsetenv("RECORDING_PATH")
		}()

		err := run()
		if err == nil {
			t.Fatalf("expected error for missing recording, got nil")
		}
	})
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("project root not found")
		}
		dir = parent
	}
}

func TestRun_SuccessWithRealRecording(t *testing.T) {
	realPath := filepath.Join(findProjectRoot(t), "data", "emp-1.tar.gz")
	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		t.Skip("emp-1.tar.gz not found, skipping run test")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_ = os.Setenv("TICK_MS", "0")
	_ = os.Setenv("AGENT_URL", server.URL)
	_ = os.Setenv("RECORDING_PATH", realPath)
	defer func() {
		_ = os.Unsetenv("TICK_MS")
		_ = os.Unsetenv("AGENT_URL")
		_ = os.Unsetenv("RECORDING_PATH")
	}()

	err := run()
	if err != nil {
		t.Fatalf("unexpected error running streamer: %v", err)
	}
}

func TestMainInvocation(t *testing.T) {
	_ = os.Setenv("TICK_MS", "invalid")
	defer func() {
		_ = os.Unsetenv("TICK_MS")
	}()

	var called bool
	oldExit := exitHandler
	exitHandler = func(_ string, _ ...any) {
		called = true
	}
	defer func() { exitHandler = oldExit }()

	main()
	if !called {
		t.Fatalf("expected exitHandler to be called in main()")
	}
}
