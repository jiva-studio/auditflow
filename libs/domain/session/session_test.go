package session

import (
	"testing"
	"time"

	"assessment/libs/domain/display"
	"assessment/libs/domain/geometry"
)

func TestTimeRange(t *testing.T) {
	now := time.Now()
	_, err := NewTimeRange(now, now.Add(-1*time.Minute))
	if err != ErrInvalidTimeRange {
		t.Fatalf("expected ErrInvalidTimeRange, got: %v", err)
	}

	tr, err := NewTimeRange(now, now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tr.TicksCount() != 3601 {
		t.Fatalf("expected 3601 ticks for 1 hour, got: %d", tr.TicksCount())
	}
	if !tr.Contains(now.Add(30 * time.Minute)) {
		t.Fatal("time range should contain midpoint")
	}
}

func TestSessionMetadata(t *testing.T) {
	tr, _ := NewTimeRange(time.Now(), time.Now().Add(1*time.Hour))
	d, _ := display.NewDisplay(0, geometry.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080}, 1.0, true)

	_, err := NewMetadata("", "emp-1", tr, MachineInfo{}, []display.Display{d})
	if err != ErrEmptySessionID {
		t.Fatalf("expected ErrEmptySessionID, got: %v", err)
	}

	_, err = NewMetadata("sess-1", "", tr, MachineInfo{}, []display.Display{d})
	if err != ErrEmptyEmployeeID {
		t.Fatalf("expected ErrEmptyEmployeeID, got: %v", err)
	}

	_, err = NewMetadata("sess-1", "emp-1", tr, MachineInfo{}, nil)
	if err != ErrNoDisplaysDefined {
		t.Fatalf("expected ErrNoDisplaysDefined, got: %v", err)
	}

	meta, err := NewMetadata("sess-1", "emp-1", tr, MachineInfo{Hostname: "DESK-1"}, []display.Display{d})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundDisp, ok := meta.FindDisplayForPoint(geometry.Point{X: 100, Y: 100})
	if !ok || foundDisp.ID != 0 {
		t.Fatal("expected display to be found for (100, 100)")
	}

	// Point outside displays
	_, ok = meta.FindDisplayForPoint(geometry.Point{X: 5000, Y: 5000})
	if ok {
		t.Fatal("expected no display found for point outside all displays")
	}

	// TicksCount with equal start and end
	now := time.Now()
	equalTR, _ := NewTimeRange(now, now)
	if equalTR.TicksCount() != 0 {
		t.Fatalf("expected 0 ticks for equal start and end, got %d", equalTR.TicksCount())
	}
}
