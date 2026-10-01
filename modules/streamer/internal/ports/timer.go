// Package ports defines the inbound and outbound interfaces for the streamer service.
package ports

import (
	"context"
	"time"
)

// Timer defines the clock pacing port for waiting between ticks.
type Timer interface {
	Wait(ctx context.Context, d time.Duration) error
}
