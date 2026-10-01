package mapper

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/libs/domain/events"
	v1 "assessment/libs/protocol/gen/go/v1"
)

func TestToDomainTickBatch(t *testing.T) {
	t.Run("nil tick batch", func(t *testing.T) {
		b := ToDomainTickBatch(nil)
		if b.Len() != 0 {
			t.Errorf("expected empty batch, got len=%d", b.Len())
		}
	})

	t.Run("valid tick batch", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Millisecond)
		pb := &v1.TickBatch{
			TickIndex: 5,
			StartTime: timestamppb.New(now),
			EndTime:   timestamppb.New(now.Add(time.Second)),
			Events: []*v1.Event{
				{
					Payload: &v1.Event_Window{
						Window: &v1.WindowEvent{
							Timestamp:   timestamppb.New(now),
							Action:      "focus",
							WindowTitle: "Test Window",
							ProcessName: "test.exe",
						},
					},
				},
				nil, // should be skipped gracefully
			},
		}

		domainBatch := ToDomainTickBatch(pb)
		if domainBatch.TickIndex != 5 {
			t.Errorf("expected tickIndex 5, got %d", domainBatch.TickIndex)
		}
		if domainBatch.Len() != 1 {
			t.Fatalf("expected 1 valid event, got %d", domainBatch.Len())
		}
		winEv, ok := domainBatch.Events()[0].(events.WindowEvent)
		if !ok || winEv.WindowTitle != "Test Window" {
			t.Errorf("unexpected event: %+v", domainBatch.Events()[0])
		}
	})
}

func TestToDomainEvent_Nil(t *testing.T) {
	if ev := ToDomainEvent(nil); ev != nil {
		t.Errorf("expected nil, got %+v", ev)
	}
	if ev := ToDomainEvent(&v1.Event{Payload: nil}); ev != nil {
		t.Errorf("expected nil, got %+v", ev)
	}
}

func TestToDomainEvent_Mouse(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	pb := &v1.Event{
		Payload: &v1.Event_Mouse{
			Mouse: &v1.MouseEvent{
				Timestamp:   timestamppb.New(now),
				Action:      "click",
				Button:      "left",
				Position:    &v1.Point{X: 10, Y: 20},
				ClickCount:  "1",
				WindowTitle: "Win",
				ProcessName: "proc.exe",
			},
		},
	}
	ev := ToDomainEvent(pb)
	mEv, ok := ev.(events.MouseEvent)
	if !ok || mEv.Action != "click" || mEv.Position.X != 10 || mEv.Position.Y != 20 {
		t.Errorf("unexpected mouse event: %+v", ev)
	}
	if nilMouse := ToDomainMouseEvent(nil); nilMouse.Action != "" {
		t.Errorf("expected empty mouse event for nil")
	}
}

func TestToDomainEvent_Clipboard(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	pb := &v1.Event{
		Payload: &v1.Event_Clipboard{
			Clipboard: &v1.ClipboardEvent{
				Timestamp: timestamppb.New(now),
				Action:    "copy",
				Text:      "INV-12345",
				Length:    9,
				SourceApp: "calc.exe",
				Formats:   []string{"text/plain"},
			},
		},
	}
	ev := ToDomainEvent(pb)
	cEv, ok := ev.(events.ClipboardEvent)
	if !ok || cEv.Text != "INV-12345" || cEv.Length != 9 {
		t.Errorf("unexpected clipboard event: %+v", ev)
	}
	if nilClip := ToDomainClipboardEvent(nil); nilClip.Text != "" {
		t.Errorf("expected empty clipboard event for nil")
	}
}

func TestToDomainEvent_OCR(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	pb := &v1.Event{
		Payload: &v1.Event_Ocr{
			Ocr: &v1.OCREvent{
				Timestamp:        timestamppb.New(now),
				Filename:         "screen.png",
				DisplayId:        1,
				Resolution:       &v1.Size{Width: 1920, Height: 1080},
				ScaleFactor:      1.5,
				WindowRect:       &v1.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080},
				DeduplicatedFrom: "prev.png",
				Blocks: []*v1.OCRTextBlock{
					{
						Text:       "Hello OCR",
						Box:        &v1.Rectangle{X: 10, Y: 10, Width: 100, Height: 20},
						Confidence: 0.99,
					},
				},
			},
		},
	}
	ev := ToDomainEvent(pb)
	oEv, ok := ev.(events.OCREvent)
	if !ok || oEv.Filename != "screen.png" || len(oEv.Blocks) != 1 || oEv.Blocks[0].Text != "Hello OCR" {
		t.Errorf("unexpected ocr event: %+v", ev)
	}
	if nilOCR := ToDomainOCREvent(nil); nilOCR.Filename != "" {
		t.Errorf("expected empty ocr event for nil")
	}
}

func TestToDomainEvent_GenericAndWindow(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	pb := &v1.Event{
		Payload: &v1.Event_Generic{
			Generic: &v1.GenericActivityEvent{
				Timestamp: timestamppb.New(now),
				Type:      "heartbeat",
			},
		},
	}
	ev := ToDomainEvent(pb)
	gEv, ok := ev.(events.GenericActivityEvent)
	if !ok || gEv.Type != "heartbeat" {
		t.Errorf("unexpected generic event: %+v", ev)
	}
	if nilGen := ToDomainGenericEvent(nil); nilGen.Type != "" {
		t.Errorf("expected empty generic event for nil")
	}
	if nilWin := ToDomainWindowEvent(nil); nilWin.WindowTitle != "" {
		t.Errorf("expected empty window event for nil")
	}
}
