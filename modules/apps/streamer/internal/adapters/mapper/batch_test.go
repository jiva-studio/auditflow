package mapper_test

import (
	"testing"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/mapper"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/geometry"
)

func TestToProtoTickBatch(t *testing.T) {
	start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Second)
	batch, err := events.NewTickBatch(5, start, end)
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	_ = batch.Add(events.MouseEvent{
		Timestamp: start.Add(100 * time.Millisecond),
		Action:    "click",
		Button:    "left",
		Position:  geometry.Point{X: 10, Y: 20},
	})
	_ = batch.Add(customUnsupportedEvent{
		ts: start.Add(200 * time.Millisecond),
	})

	pb := mapper.ToProtoTickBatch(batch)
	if pb.TickIndex != 5 {
		t.Errorf("got tick index %d, want 5", pb.TickIndex)
	}
	if len(pb.Events) != 1 {
		t.Errorf("got %d events in protobuf batch, want 1", len(pb.Events))
	}
}

func TestToProtoTickBatch_PreRollTick0(t *testing.T) {
	start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Second)
	batch, err := events.NewTickBatch(0, start, end)
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	// Pre-roll event before started_at
	if err := batch.Add(events.WindowEvent{
		Timestamp:   start.Add(-500 * time.Millisecond),
		Action:      "focus_change",
		WindowTitle: "Initial Window",
		ProcessName: "init.exe",
	}); err != nil {
		t.Fatalf("failed to add pre-roll event: %v", err)
	}

	if err := batch.Add(events.MouseEvent{
		Timestamp: start.Add(100 * time.Millisecond),
		Action:    "click",
		Button:    "left",
	}); err != nil {
		t.Fatalf("failed to add normal event: %v", err)
	}

	pb := mapper.ToProtoTickBatch(batch)
	if pb.TickIndex != 0 {
		t.Errorf("got tick index %d, want 0", pb.TickIndex)
	}
	if len(pb.Events) != 2 {
		t.Fatalf("got %d events in protobuf batch, want 2", len(pb.Events))
	}
	if pb.Events[0].GetWindow().GetWindowTitle() != "Initial Window" {
		t.Errorf("expected initial window as first event, got %v", pb.Events[0].GetWindow())
	}
}
