package service_test

import (
	"context"
	"errors"
	"testing"

	"assessment/modules/apps/server/internal/service"
	"assessment/modules/libs/domain/audit"
)

type mockRepo struct {
	saved      []audit.Popup
	saveErr    error
	findPopups []audit.Popup
	findErr    error
	lastFilter audit.Filter
}

func (m *mockRepo) Save(_ context.Context, popup audit.Popup) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, popup)
	return nil
}

func (m *mockRepo) Find(_ context.Context, filter audit.Filter) ([]audit.Popup, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	m.lastFilter = filter
	return m.findPopups, nil
}

func TestNewService_Validation(t *testing.T) {
	_, err := service.NewService(nil)
	if !errors.Is(err, service.ErrNilRepository) {
		t.Fatalf("expected ErrNilRepository, got %v", err)
	}

	repo := &mockRepo{}
	s, err := service.NewService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil Service")
	}
}

func TestService_RecordPopup(t *testing.T) {
	repo := &mockRepo{}
	s, err := service.NewService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	popup, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title", "Body")
	ctx := context.Background()

	if err := s.RecordPopup(ctx, popup); err != nil {
		t.Fatalf("unexpected error recording popup: %v", err)
	}

	if len(repo.saved) != 1 || repo.saved[0].Employee != "emp-1" {
		t.Fatalf("popup was not saved to repository as expected")
	}

	expectedErr := errors.New("save failed")
	repo.saveErr = expectedErr
	if err := s.RecordPopup(ctx, popup); !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestService_QueryPopups(t *testing.T) {
	p1, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title 1", "Body 1")
	p2, _ := audit.NewPopup("emp-2", "rule-2", "2026-03-10T13:00:00Z", "Title 2", "Body 2")
	repo := &mockRepo{
		findPopups: []audit.Popup{p1, p2},
	}
	s, err := service.NewService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	filter := audit.Filter{Employee: "emp-1"}

	results, err := s.QueryPopups(ctx, filter)
	if err != nil {
		t.Fatalf("unexpected error querying popups: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if repo.lastFilter.Employee != "emp-1" {
		t.Fatalf("expected filter employee to be passed to repo")
	}

	expectedErr := errors.New("find failed")
	repo.findErr = expectedErr
	if _, err := s.QueryPopups(ctx, filter); !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}
