package timer_test

import (
	"context"
	"testing"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/timer"
)

func TestMockTimer(t *testing.T) {
	mt := timer.NewMockTimer()
	if err := mt.Wait(context.Background(), 100*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	waits := mt.WaitedDurations
	if len(waits) != 1 || waits[0] != 100*time.Millisecond {
		t.Errorf("got waits %v, want [100ms]", waits)
	}

	mt.WaitedDurations = nil
	if len(mt.WaitedDurations) != 0 {
		t.Errorf("expected 0 waits after Reset, got %d", len(mt.WaitedDurations))
	}
}
