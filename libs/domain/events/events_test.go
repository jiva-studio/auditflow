package events

import (
	"testing"
	"time"

	"assessment/libs/domain/geometry"
)

func TestEvents(t *testing.T) {
	now := time.Now()

	wEvent := WindowEvent{
		Timestamp:   now,
		Action:      "focus_change",
		WindowTitle: "Outlook",
		ProcessName: "OUTLOOK.EXE",
	}
	if wEvent.GetType() != EventTypeWindow || !wEvent.GetTimestamp().Equal(now) {
		t.Fatal("invalid window event")
	}

	mEvent := MouseEvent{
		Timestamp: now,
		Action:    "click",
		Position:  geometry.Point{X: 100, Y: 200},
	}
	if mEvent.GetType() != EventTypeMouse {
		t.Fatal("invalid mouse event")
	}

	cEvent := ClipboardEvent{
		Timestamp: now,
		Text:      "INV-123",
	}
	if cEvent.GetType() != EventTypeClipboard {
		t.Fatal("invalid clipboard event")
	}

	oEvent := OCREvent{
		Timestamp: now,
		DisplayID: 1,
	}
	if oEvent.GetType() != EventTypeOCR {
		t.Fatal("invalid OCR event")
	}

	batch := TickBatch{
		TickIndex: 0,
		StartTime: now,
		EndTime:   now.Add(1 * time.Second),
		Events:    []Event{wEvent, mEvent, cEvent, oEvent},
	}
	if len(batch.Events) != 4 {
		t.Fatalf("expected 4 events in batch, got %d", len(batch.Events))
	}
}
