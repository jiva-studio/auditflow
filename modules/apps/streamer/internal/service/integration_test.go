package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/client"
	"assessment/modules/apps/streamer/internal/adapters/source"
	"assessment/modules/apps/streamer/internal/adapters/timer"
	"assessment/modules/apps/streamer/internal/service"
)

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

func TestStreamer_RealRecordingIntegration(t *testing.T) {
	recordingPath := filepath.Join(findProjectRoot(t), "data", "emp-1.tar.gz")
	if _, err := os.Stat(recordingPath); os.IsNotExist(err) {
		t.Skip("emp-1.tar.gz not found, skipping integration test")
	}

	tickCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tickCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	src := source.NewTarGzEventSource(recordingPath, nil)
	c, err := client.NewHTTPClient(server.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("init client failed: %v", err)
	}

	// Use mock timer with zero delay for fast replay in test
	tm := timer.NewMockTimer()
	svc, err := service.NewReplayService(src, c, tm, 0)
	if err != nil {
		t.Fatalf("init service failed: %v", err)
	}

	meta, count, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("service run failed: %v", err)
	}

	if meta.EmployeeID != "emp-1" {
		t.Errorf("got employee %s, want emp-1", meta.EmployeeID)
	}
	if count == 0 {
		t.Errorf("expected dispatched ticks > 0, got 0")
	}
	if tickCount != count {
		t.Errorf("server received %d ticks, service reported %d", tickCount, count)
	}
}
