// Package ports defines the inbound and outbound interfaces for the streamer service.
package ports

import (
	"context"

	"assessment/libs/domain/events"
)

// AgentClient defines the outbound port for dispatching tick batches to an agent.
type AgentClient interface {
	SendTick(ctx context.Context, batch events.TickBatch) error
}
