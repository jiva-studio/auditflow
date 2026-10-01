package timer_test

import (
	"context"
	"testing"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/timer"
)

func TestRealTimer_Wait(t *testing.T) {
	t.Run("normal duration", func(t *testing.T) {
		tm := timer.NewRealTimer()
		start := time.Now()
		err := tm.Wait(context.Background(), 10*time.Millisecond)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if time.Since(start) < 9*time.Millisecond {
			t.Errorf("timer finished too quickly: %v", time.Since(start))
		}
	})

	t.Run("zero duration without cancel", func(t *testing.T) {
		tm := timer.NewRealTimer()
		ctx := context.Background()
		err := tm.Wait(ctx, 0)
		if err != nil {
			t.Fatalf("expected nil error for 0 duration with active ctx, got %v", err)
		}

		allocs := testing.AllocsPerRun(100, func() {
			_ = tm.Wait(ctx, 0)
		})
		if allocs > 0 {
			t.Errorf("expected 0 allocations for d=0, got %f", allocs)
		}
	})

	t.Run("zero duration with cancelled ctx", func(t *testing.T) {
		tm := timer.NewRealTimer()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := tm.Wait(ctx, 0)
		if err == nil {
			t.Fatalf("expected error for cancelled ctx with 0 duration, got nil")
		}
	})

	t.Run("negative duration", func(t *testing.T) {
		tm := timer.NewRealTimer()
		err := tm.Wait(context.Background(), -5*time.Second)
		if err != nil {
			t.Fatalf("expected nil error for negative duration with active ctx, got %v", err)
		}
	})

	t.Run("context cancelled during wait", func(t *testing.T) {
		tm := timer.NewRealTimer()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := tm.Wait(ctx, 1*time.Second)
		if err == nil {
			t.Fatalf("expected context error, got nil")
		}
	})
}
