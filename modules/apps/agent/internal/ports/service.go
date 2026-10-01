package ports

import (
	"context"

	"assessment/modules/libs/domain/events"
)

// AgentService is the driving inbound port for processing telemetry tick batches.
type AgentService interface {
	ProcessTick(ctx context.Context, batch events.TickBatch) error
	CheckReadiness(ctx context.Context) error
}
