package testutil_test

import (
	"os/exec"
	"testing"
	"time"

	"assessment/tests/e2e/testutil"
)

func TestMemoryTracker_Lifecycle(t *testing.T) {
	cmd := exec.Command("sleep", "0.05")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	tracker := testutil.StartMemoryTracker(cmd)
	time.Sleep(20 * time.Millisecond)

	if err := cmd.Wait(); err != nil {
		t.Fatalf("command wait failed: %v", err)
	}

	peakRSS := tracker.Stop()
	if peakRSS < 0 {
		t.Errorf("peak RSS should be non-negative, got %f", peakRSS)
	}
}

func TestMemoryTracker_NilOrUnstartedCmd(t *testing.T) {
	cmd := exec.Command("echo", "hello")
	tracker := testutil.StartMemoryTracker(cmd)
	peak := tracker.Stop()
	if peak < 0 {
		t.Errorf("peak RSS for unstarted command should be non-negative, got %f", peak)
	}
}
