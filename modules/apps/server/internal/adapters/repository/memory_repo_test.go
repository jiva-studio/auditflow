package repository_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"assessment/modules/apps/server/internal/adapters/repository"
	"assessment/modules/libs/domain/audit"
)

func populateRepo(t *testing.T) (*repository.MemoryRepository, context.Context) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	ctx := context.Background()

	p1, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title 1", "Body 1")
	p2, _ := audit.NewPopup("emp-2", "rule-2", "2026-03-10T13:00:00Z", "Title 2", "Body 2")
	p3, _ := audit.NewPopup("emp-1", "rule-2", "2026-03-10T14:00:00Z", "Title 3", "Body 3")

	for _, p := range []audit.Popup{p1, p2, p3} {
		if err := repo.Save(ctx, p); err != nil {
			t.Fatalf("unexpected save error: %v", err)
		}
	}
	return repo, ctx
}

func TestMemoryRepository_FindAll(t *testing.T) {
	repo, ctx := populateRepo(t)
	all, err := repo.Find(ctx, audit.Filter{})
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 popups, got %d", len(all))
	}
}

func TestMemoryRepository_FindByEmployee(t *testing.T) {
	repo, ctx := populateRepo(t)
	emp1Popups, err := repo.Find(ctx, audit.Filter{Employee: "emp-1"})
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	if len(emp1Popups) != 2 {
		t.Fatalf("expected 2 popups for emp-1, got %d", len(emp1Popups))
	}
}

func TestMemoryRepository_FindByRule(t *testing.T) {
	repo, ctx := populateRepo(t)
	rule2Popups, err := repo.Find(ctx, audit.Filter{Rule: "rule-2"})
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	if len(rule2Popups) != 2 {
		t.Fatalf("expected 2 popups for rule-2, got %d", len(rule2Popups))
	}
}

func TestMemoryRepository_FindByEmployeeAndRule(t *testing.T) {
	repo, ctx := populateRepo(t)
	emp1Rule2, err := repo.Find(ctx, audit.Filter{Employee: "emp-1", Rule: "rule-2"})
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	if len(emp1Rule2) != 1 || emp1Rule2[0].Title != "Title 3" {
		t.Fatalf("expected 1 popup with Title 3, got %v", emp1Rule2)
	}
}

func TestMemoryRepository_ContextCancellation(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	popup, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title", "Body")
	if err := repo.Save(ctx, popup); err == nil {
		t.Fatal("expected error on cancelled context during Save")
	}

	if _, err := repo.Find(ctx, audit.Filter{}); err == nil {
		t.Fatal("expected error on cancelled context during Find")
	}
}

func TestMemoryRepository_ConcurrentAccess(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()

	const numWorkers = 20
	const itemsPerWorker = 50

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		w := w
		go func() {
			defer wg.Done()
			empID := fmt.Sprintf("emp-%d", w%5)
			for i := 0; i < itemsPerWorker; i++ {
				p, _ := audit.NewPopup(empID, "rule-test", "2026-03-10T12:00:00Z", "Title", "Body")
				_ = repo.Save(ctx, p)
				_, _ = repo.Find(ctx, audit.Filter{Employee: empID})
			}
		}()
	}

	wg.Wait()

	all, err := repo.Find(ctx, audit.Filter{})
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	expectedTotal := numWorkers * itemsPerWorker
	if len(all) != expectedTotal {
		t.Fatalf("expected %d total items, got %d", expectedTotal, len(all))
	}
}

func TestMemoryRepository_FindEmpty(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()

	results, err := repo.Find(ctx, audit.Filter{Employee: "non-existent"})
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	if results == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
