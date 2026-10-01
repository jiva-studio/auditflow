// Package ports defines the inbound and outbound interfaces for the agent service.
package ports

import (
	"context"

	"assessment/libs/domain/audit"
)

// AuditClient is the outbound port for dispatching triggered audit popups to the central server.
type AuditClient interface {
	SendPopup(ctx context.Context, popup audit.Popup) error
}
