// Package timer provides clock timer implementations for pacing playback.
package timer

import (
	"context"
	"time"
)

// RealTimer implements ports.Timer using real system timers.
type RealTimer struct{}

// NewRealTimer constructs a new RealTimer.
func NewRealTimer() *RealTimer {
	return &RealTimer{}
}

// Wait pauses execution for duration d or until ctx is cancelled.
func (r *RealTimer) Wait(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}

	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
