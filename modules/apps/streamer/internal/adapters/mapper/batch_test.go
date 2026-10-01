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
