// Package timer provides clock timer implementations for pacing playback.
package timer

import (
	"context"
	"time"
)

// MockTimer implements ports.Timer for instant deterministic tests.
type MockTimer struct {
	WaitedDurations []time.Duration
}

// NewMockTimer constructs a new MockTimer.
func NewMockTimer() *MockTimer {
	return &MockTimer{
		WaitedDurations: make([]time.Duration, 0),
	}
}

// Wait records the duration and immediately returns ctx.Err() if cancelled, nil otherwise.
func (m *MockTimer) Wait(ctx context.Context, d time.Duration) error {
	m.WaitedDurations = append(m.WaitedDurations, d)
	return ctx.Err()
}
