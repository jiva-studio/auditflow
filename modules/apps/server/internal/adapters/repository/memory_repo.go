// Package repository provides storage adapters for the audit server.
package repository

import (
	"context"
	"sync"

	"assessment/modules/apps/server/internal/ports"
	"assessment/modules/libs/domain/audit"
)

// MemoryRepository implements ports.AuditRepository in-memory with thread-safety.
type MemoryRepository struct {
	mu     sync.RWMutex
	popups []audit.Popup
}

// NewMemoryRepository constructs a new empty MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		popups: make([]audit.Popup, 0),
	}
}

var _ ports.AuditRepository = (*MemoryRepository)(nil)

// Save appends a popup to the repository.
func (r *MemoryRepository) Save(ctx context.Context, popup audit.Popup) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.popups = append(r.popups, popup)
	return nil
}

// Find retrieves popups matching the filter.
func (r *MemoryRepository) Find(ctx context.Context, filter audit.Filter) ([]audit.Popup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]audit.Popup, 0)
	for _, p := range r.popups {
		if filter.Matches(p) {
			matched = append(matched, p)
		}
	}

	return matched, nil
}
