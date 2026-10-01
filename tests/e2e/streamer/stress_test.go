package streamer_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"assessment/tests/e2e/testutil"
)

func TestStreamer_E2E_Stress_MemoryBounded(t *testing.T) {
	binPath := testutil.GetStreamerBin(t)

	// Configurable stress test parameters with sensible fast defaults:
	// - STRESS_SECONDS: duration of recording in seconds (default: 300)
	// - STRESS_EPS: events per second (default: 500)
	// - STRESS_MEM_LIMIT: GOMEMLIMIT soft limit (default: 30MiB)
	// - STRESS_MAX_RSS_MB: budget threshold for peak RSS (default: 50.0 MB)
	numSeconds := testutil.GetEnvInt("STRESS_SECONDS", 300)
	eventsPerSec := testutil.GetEnvInt("STRESS_EPS", 500)
	memLimit := testutil.GetEnvString("STRESS_MEM_LIMIT", "30MiB")
	maxAllowedRSSMB := testutil.GetEnvFloat("STRESS_MAX_RSS_MB", 50.0)

	totalExpectedEvents := numSeconds * eventsPerSec

	tmpArchive := filepath.Join(t.TempDir(), "stress_recording.tar.gz")

	genStart := time.Now()
	fileSizeBytes, err := testutil.GenerateSyntheticRecording(t, tmpArchive, numSeconds, eventsPerSec)
	if err != nil {
		t.Fatalf("failed to generate synthetic recording: %v", err)
	}
	genDuration := time.Since(genStart)

	fileSizeMB := float64(fileSizeBytes) / (1024 * 1024)
	t.Logf("Generated synthetic stress archive: %.2f MB (%d bytes) in %v",
		fileSizeMB, fileSizeBytes, genDuration)
	t.Logf("Dataset scale: %d events across %d seconds (%d EPS, GOMEMLIMIT=%s)",
		totalExpectedEvents, numSeconds, eventsPerSec, memLimit)

	var totalTicks atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-protobuf" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		totalTicks.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"RECORDING_PATH="+tmpArchive,
		"AGENT_URL="+server.URL,
		"TICK_MS=0",
		"GOMEMLIMIT="+memLimit,
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start streamer process: %v", err)
	}

	tracker := testutil.StartMemoryTracker(cmd)
	streamStart := time.Now()

	err = cmd.Wait()
	streamDuration := time.Since(streamStart)
	peakRSSMB := tracker.Stop()

	if err != nil {
		t.Fatalf("streamer process execution failed: %v", err)
	}

	if int(totalTicks.Load()) != numSeconds {
		t.Errorf("total ticks mismatch: got %d, want %d", totalTicks.Load(), numSeconds)
	}

	eventsPerSecThroughput := float64(totalExpectedEvents) / streamDuration.Seconds()
	t.Logf("================ STRESS TEST RESULTS ================")
	t.Logf("  - Recording on disk:       %.2f MB (.tar.gz compressed)", fileSizeMB)
	t.Logf("  - Total events streamed:   %d events in %d ticks", totalExpectedEvents, totalTicks.Load())
	t.Logf("  - Total streaming time:    %v", streamDuration)
	t.Logf("  - Streamer throughput:     %.0f events/second", eventsPerSecThroughput)
	t.Logf("  - Process Peak RSS:        %.2f MB (GOMEMLIMIT=%s enforced)", peakRSSMB, memLimit)
	t.Logf("=====================================================")

	// Verify bounded memory usage
	if peakRSSMB > maxAllowedRSSMB {
		t.Errorf("streamer process memory exceeded budget: peak RSS %.2f MB > %.2f MB", peakRSSMB, maxAllowedRSSMB)
	}
}
