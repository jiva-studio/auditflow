package streamer_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

type GoldenFixture struct {
	TotalTicks      int    `json:"total_ticks"`
	StreamSHA256    string `json:"stream_sha256"`
	FirstTickSHA256 string `json:"first_tick_sha256"`
	LastTickSHA256  string `json:"last_tick_sha256"`
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("project root not found (.git missing in parent hierarchy)")
		}
		dir = parent
	}
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
	root := findProjectRoot(t)

	binPath := os.Getenv("STREAMER_BIN")
	if binPath == "" {
		binPath = filepath.Join(root, "bin", "streamer")
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(root, "data")
	}

	tests := []string{"emp-1", "emp-2", "emp-3"}

	for _, name := range tests {
		name := name
		t.Run(name, func(t *testing.T) {
			fixture := loadFixture(t, name+".json")

			var mu sync.Mutex
			var totalTicks atomic.Int32
			var firstBody, lastBody []byte
			hasher := sha256.New()

			// Pure Black-Box Receiver: collects raw payload and counts requests
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-protobuf" {
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				mu.Lock()
				if totalTicks.Load() == 0 {
					firstBody = body
				}
				lastBody = body
				hasher.Write(body)
				mu.Unlock()

				totalTicks.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			archivePath := filepath.Join(dataDir, name+".tar.gz")

			// Execute streamer binary
			cmd := exec.Command(binPath)
			cmd.Env = append(os.Environ(),
				"RECORDING_PATH="+archivePath,
				"AGENT_URL="+server.URL,
				"TICK_MS=0",
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("streamer execution failed (exit code non-zero): %v\noutput: %s", err, output)
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
