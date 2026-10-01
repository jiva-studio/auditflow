package streamer_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

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

	tests := []string{"emp-1", "emp-2", "emp-3"}

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
