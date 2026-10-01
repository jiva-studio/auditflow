package streamer_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"assessment/tests/e2e/testutil"
)

type GoldenFixture struct {
	TotalTicks      int    `json:"total_ticks"`
	StreamSHA256    string `json:"stream_sha256"`
	FirstTickSHA256 string `json:"first_tick_sha256"`
	LastTickSHA256  string `json:"last_tick_sha256"`
}

func loadFixture(t *testing.T, fixtureName string) GoldenFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("fixtures", fixtureName))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixtureName, err)
	}
	var f GoldenFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("failed to parse fixture %s: %v", fixtureName, err)
	}
	return f
}

func TestStreamer_E2E(t *testing.T) {
	binPath := testutil.GetStreamerBin(t)
	dataDir := testutil.GetDataDir(t)

	tests := []string{"emp-1", "emp-2", "emp-3", "emp-synthetic", "emp-edge", "emp-multidisplay", "emp-quadhd", "emp-contextreset"}

	for _, name := range tests {
		name := name
		t.Run(name, func(t *testing.T) {
			fixture := loadFixture(t, name+".json")

			var mu sync.Mutex
			var totalTicks atomic.Int32
			var firstBody, lastBody []byte
			hasher := sha256.New()

			server := testutil.NewMockAgentServer(t, func(body []byte) {
				mu.Lock()
				if totalTicks.Load() == 0 {
					firstBody = body
				}
				lastBody = body
				hasher.Write(body)
				mu.Unlock()
				totalTicks.Add(1)
			})
			defer server.Close()

			archivePath := filepath.Join(dataDir, name+".tar.gz")
			output, err := testutil.RunStreamer(t, binPath, archivePath, server.URL)
			if err != nil {
				t.Fatalf("streamer execution failed: %v\noutput: %s", err, output)
			}

			// 1. Validate total ticks received
			if int(totalTicks.Load()) != fixture.TotalTicks {
				t.Errorf("total ticks mismatch: got %d, want %d", totalTicks.Load(), fixture.TotalTicks)
			}

			// 2. Validate deterministic full stream digest
			actualStreamHash := hex.EncodeToString(hasher.Sum(nil))
			if actualStreamHash != fixture.StreamSHA256 {
				t.Errorf("stream sha256 mismatch:\ngot  %s\nwant %s", actualStreamHash, fixture.StreamSHA256)
			}

			// 3. Validate first and last tick hashes
			firstHash := sha256.Sum256(firstBody)
			lastHash := sha256.Sum256(lastBody)

			actualFirstHash := hex.EncodeToString(firstHash[:])
			if actualFirstHash != fixture.FirstTickSHA256 {
				t.Errorf("first tick sha256 mismatch:\ngot  %s\nwant %s", actualFirstHash, fixture.FirstTickSHA256)
			}

			actualLastHash := hex.EncodeToString(lastHash[:])
			if actualLastHash != fixture.LastTickSHA256 {
				t.Errorf("last tick sha256 mismatch:\ngot  %s\nwant %s", actualLastHash, fixture.LastTickSHA256)
			}
		})
	}
}

func TestStreamer_E2E_GracefulShutdown(t *testing.T) {
	binPath := testutil.GetStreamerBin(t)
	dataDir := testutil.GetDataDir(t)
	archivePath := filepath.Join(dataDir, "emp-1.tar.gz")

	var totalTicks atomic.Int32
	server := testutil.NewMockAgentServer(t, func(body []byte) {
		totalTicks.Add(1)
	})
	defer server.Close()

	// Slow down ticks (100ms each) so we can cleanly interrupt in-flight
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"RECORDING_PATH="+archivePath,
		"AGENT_URL="+server.URL,
		"TICK_MS=100",
	)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start streamer process: %v", err)
	}

	// Wait until at least 3 ticks are received
	deadline := time.Now().Add(5 * time.Second)
	for totalTicks.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if totalTicks.Load() < 3 {
		t.Fatalf("streamer did not produce ticks in time, got %d", totalTicks.Load())
	}

	// Send SIGTERM to test graceful cancellation
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	err := cmd.Wait()
	if err != nil {
		t.Fatalf("streamer process failed to exit cleanly on SIGTERM: %v\noutput: %s", err, outBuf.String())
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "replay cancelled after") {
		t.Errorf("expected graceful cancellation message in logs, got: %s", outStr)
	}
}
