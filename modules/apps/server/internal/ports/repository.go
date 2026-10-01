// Package ports defines the inbound and outbound interfaces for the central audit server.
package ports

import (
	"context"

	"assessment/modules/libs/domain/audit"
)

// AuditRepository defines the persistence interface for storing and querying popups.
type AuditRepository interface {
	Save(ctx context.Context, popup audit.Popup) error
	Find(ctx context.Context, filter audit.Filter) ([]audit.Popup, error)
}
