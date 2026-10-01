package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	key := "AGENT_TEST_ENV_VAR"
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
	t.Run("invalid SERVER_TIMEOUT_MS", func(t *testing.T) {
		_ = os.Setenv("SERVER_TIMEOUT_MS", "invalid")
		defer func() { _ = os.Unsetenv("SERVER_TIMEOUT_MS") }()

		if err := run(); err == nil {
			t.Fatal("expected error for invalid timeout")
		}
	})

	t.Run("missing rules file", func(t *testing.T) {
		_ = os.Setenv("SERVER_TIMEOUT_MS", "1000")
		_ = os.Setenv("RULES_PATH", "/non/existent/rules.json")
		defer func() {
			_ = os.Unsetenv("SERVER_TIMEOUT_MS")
			_ = os.Unsetenv("RULES_PATH")
		}()

		if err := run(); err == nil {
			t.Fatal("expected error for missing rules")
		}
	})

	t.Run("empty SERVER_URL", func(t *testing.T) {
		tmpDir := t.TempDir()
		rulesFile := filepath.Join(tmpDir, "rules.json")
		_ = os.WriteFile(rulesFile, []byte(`{"rules":[]}`), 0600)

		_ = os.Setenv("SERVER_TIMEOUT_MS", "1000")
		_ = os.Setenv("RULES_PATH", rulesFile)
		_ = os.Setenv("SERVER_URL", "   ")
		defer func() {
			_ = os.Unsetenv("SERVER_TIMEOUT_MS")
			_ = os.Unsetenv("RULES_PATH")
			_ = os.Unsetenv("SERVER_URL")
		}()

		if err := run(); err == nil {
			t.Fatal("expected error for empty server url")
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

func TestRun_GracefulShutdown(t *testing.T) {
	realRulesPath := filepath.Join(findProjectRoot(t), "data", "rules.json")
	if _, err := os.Stat(realRulesPath); os.IsNotExist(err) {
		t.Skip("rules.json not found, skipping run test")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_ = os.Setenv("PORT", "0") // random available port
	_ = os.Setenv("RULES_PATH", realRulesPath)
	_ = os.Setenv("SERVER_URL", server.URL)
	_ = os.Setenv("EMPLOYEE_ID", "emp-test")
	_ = os.Setenv("SERVER_TIMEOUT_MS", "1000")
	defer func() {
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("RULES_PATH")
		_ = os.Unsetenv("SERVER_URL")
		_ = os.Unsetenv("EMPLOYEE_ID")
		_ = os.Unsetenv("SERVER_TIMEOUT_MS")
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run()
	}()

	// Wait for server to start up, then send SIGTERM
	time.Sleep(100 * time.Millisecond)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run() returned error on graceful shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for graceful shutdown")
	}
}

func TestMainInvocation(t *testing.T) {
	_ = os.Setenv("SERVER_TIMEOUT_MS", "invalid")
	defer func() {
		_ = os.Unsetenv("SERVER_TIMEOUT_MS")
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
