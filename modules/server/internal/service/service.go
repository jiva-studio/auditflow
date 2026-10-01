// Package service implements the business logic orchestration for the central audit server.
package service

import (
	"context"
	"errors"

	"assessment/libs/domain/audit"
	"assessment/modules/server/internal/ports"
)

// Sentinel errors for Service.
var (
	ErrNilRepository = errors.New("audit repository cannot be nil")
)

// Service coordinates popup recording and retrieval via an AuditRepository.
type Service struct {
	repo ports.AuditRepository
}

// NewService constructs and validates a new server Service instance.
func NewService(repo ports.AuditRepository) (*Service, error) {
	if repo == nil {
		return nil, ErrNilRepository
	}
	return &Service{repo: repo}, nil
}

var _ ports.ServerService = (*Service)(nil)

// RecordPopup saves a popup into the repository.
func (s *Service) RecordPopup(ctx context.Context, popup audit.Popup) error {
	return s.repo.Save(ctx, popup)
}

// QueryPopups queries the repository for popups matching the specified filter.
func (s *Service) QueryPopups(ctx context.Context, filter audit.Filter) ([]audit.Popup, error) {
	return s.repo.Find(ctx, filter)
}
