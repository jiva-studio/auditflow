package events_test

import (
	"errors"
	"testing"
	"time"

	"assessment/libs/domain/events"
	"assessment/libs/domain/geometry"
)

func TestEvents_ImplementsInterface(t *testing.T) {
	now := time.Now()

	wEvent := events.WindowEvent{
		Timestamp:   now,
		Action:      "focus_change",
		WindowTitle: "Outlook",
		ProcessName: "OUTLOOK.EXE",
	}
	if wEvent.GetType() != events.EventTypeWindow || !wEvent.GetTimestamp().Equal(now) {
		t.Fatal("invalid window event")
	}

	mEvent := events.MouseEvent{
		Timestamp: now,
		Action:    "click",
		Position:  geometry.Point{X: 100, Y: 200},
	}
	if mEvent.GetType() != events.EventTypeMouse {
		t.Fatal("invalid mouse event")
	}

	cEvent := events.ClipboardEvent{
		Timestamp: now,
		Text:      "INV-123",
	}
	if cEvent.GetType() != events.EventTypeClipboard {
		t.Fatal("invalid clipboard event")
	}

	oEvent := events.OCREvent{
		Timestamp: now,
		DisplayID: 1,
	}
	if oEvent.GetType() != events.EventTypeOCR || !oEvent.GetTimestamp().Equal(now) {
		t.Fatal("invalid OCR event")
	}

	gEvent := events.GenericActivityEvent{
		Timestamp: now,
		Type:      events.EventTypeKeystroke,
	}
	if gEvent.GetType() != events.EventTypeKeystroke || !gEvent.GetTimestamp().Equal(now) {
		t.Fatal("invalid generic activity event")
	}
}

func TestTickBatch_ValidCreation(t *testing.T) {
	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(1 * time.Second)

	batch, err := events.NewTickBatch(0, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if batch.TickIndex != 0 || !batch.StartTime.Equal(start) || !batch.EndTime.Equal(end) {
		t.Fatalf("batch mismatch: %+v", batch)
	}
	if !batch.IsEmpty() || batch.Len() != 0 {
		t.Fatalf("expected empty batch")
	}
}

func TestTickBatch_InvalidParams(t *testing.T) {
	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(1 * time.Second)

	if _, err := events.NewTickBatch(-1, start, end); !errors.Is(err, events.ErrInvalidTickIndex) {
		t.Errorf("expected ErrInvalidTickIndex, got %v", err)
	}
	if _, err := events.NewTickBatch(0, time.Time{}, end); !errors.Is(err, events.ErrInvalidInterval) {
		t.Errorf("expected ErrInvalidInterval for zero start, got %v", err)
	}
	if _, err := events.NewTickBatch(0, start, time.Time{}); !errors.Is(err, events.ErrInvalidInterval) {
		t.Errorf("expected ErrInvalidInterval for zero end, got %v", err)
	}
	if _, err := events.NewTickBatch(0, end, start); !errors.Is(err, events.ErrInvalidInterval) {
		t.Errorf("expected ErrInvalidInterval for end before start, got %v", err)
	}
	if _, err := events.NewTickBatch(0, start, start); !errors.Is(err, events.ErrInvalidInterval) {
		t.Errorf("expected ErrInvalidInterval for equal times, got %v", err)
	}
}

func TestTickBatch_AddAndOrder(t *testing.T) {
	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(1 * time.Second)

	batch, err := events.NewTickBatch(1, start, end)
	if err != nil {
		t.Fatalf("failed to create batch: %v", err)
	}

	ev3 := events.MouseEvent{Timestamp: start.Add(500 * time.Millisecond), Action: "click"}
	ev1 := events.WindowEvent{Timestamp: start.Add(100 * time.Millisecond), Action: "focus_change"}
	ev2 := events.ClipboardEvent{Timestamp: start.Add(200 * time.Millisecond), Text: "data"}

	_ = batch.Add(ev3)
	_ = batch.Add(ev1)
	_ = batch.Add(ev2)

	if batch.Len() != 3 || batch.IsEmpty() {
		t.Fatalf("expected len 3, got %d", batch.Len())
	}

	evList := batch.Events()
	assertEventOrder(t, evList, start)
}

func assertEventOrder(t *testing.T, evList []events.Event, start time.Time) {
	t.Helper()
	if len(evList) != 3 {
		t.Fatalf("expected 3 events, got %d", len(evList))
	}
	if evList[0].GetType() != events.EventTypeWindow || !evList[0].GetTimestamp().Equal(start.Add(100*time.Millisecond)) {
		t.Errorf("evList[0] mismatch: %v at %v", evList[0].GetType(), evList[0].GetTimestamp())
	}
	if evList[1].GetType() != events.EventTypeClipboard || !evList[1].GetTimestamp().Equal(start.Add(200*time.Millisecond)) {
		t.Errorf("evList[1] mismatch: %v at %v", evList[1].GetType(), evList[1].GetTimestamp())
	}
	if evList[2].GetType() != events.EventTypeMouse || !evList[2].GetTimestamp().Equal(start.Add(500*time.Millisecond)) {
		t.Errorf("evList[2] mismatch: %v at %v", evList[2].GetType(), evList[2].GetTimestamp())
	}
}

func TestTickBatch_BoundaryErrors(t *testing.T) {
	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(1 * time.Second)

	batch, err := events.NewTickBatch(0, start, end)
	if err != nil {
		t.Fatalf("failed to create batch: %v", err)
	}

	// Nil event
	if err := batch.Add(nil); !errors.Is(err, events.ErrNilEvent) {
		t.Fatalf("expected ErrNilEvent, got %v", err)
	}

	// Event before start
	evBefore := events.WindowEvent{Timestamp: start.Add(-1 * time.Millisecond)}
	if err := batch.Add(evBefore); !errors.Is(err, events.ErrEventOutOfBounds) {
		t.Fatalf("expected ErrEventOutOfBounds for before start, got %v", err)
	}

	// Event exactly at end (exclusive upper bound)
	evAtEnd := events.WindowEvent{Timestamp: end}
	if err := batch.Add(evAtEnd); !errors.Is(err, events.ErrEventOutOfBounds) {
		t.Fatalf("expected ErrEventOutOfBounds for event at end, got %v", err)
	}

	// Event after end
	evAfter := events.WindowEvent{Timestamp: end.Add(10 * time.Millisecond)}
	if err := batch.Add(evAfter); !errors.Is(err, events.ErrEventOutOfBounds) {
		t.Fatalf("expected ErrEventOutOfBounds for event after end, got %v", err)
	}

	// Event exactly at start (inclusive lower bound)
	evAtStart := events.WindowEvent{Timestamp: start}
	if err := batch.Add(evAtStart); err != nil {
		t.Fatalf("expected event at start to succeed, got %v", err)
	}
}
