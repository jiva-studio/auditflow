// Package ports defines the inbound and outbound interfaces for the streamer service.
package ports

import (
	"context"

	"assessment/libs/domain/events"
	"assessment/libs/domain/session"
)

// EventSource defines the inbound port for reading session metadata and tick streams.
type EventSource interface {
	LoadMetadata(ctx context.Context) (session.Metadata, error)
	StreamTicks(ctx context.Context) (<-chan events.TickBatch, <-chan error, error)
}
