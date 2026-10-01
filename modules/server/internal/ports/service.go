package ports

import (
	"context"

	"assessment/libs/domain/audit"
)

// ServerService defines the business logic contract for the central audit server.
type ServerService interface {
	RecordPopup(ctx context.Context, popup audit.Popup) error
	QueryPopups(ctx context.Context, filter audit.Filter) ([]audit.Popup, error)
}
