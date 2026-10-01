package mapper

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/modules/libs/domain/events"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

func TestToDomainTickBatch_Nil(t *testing.T) {
	b := ToDomainTickBatch(nil)
	if b.Len() != 0 {
		t.Errorf("expected empty batch, got len=%d", b.Len())
	}
}

func TestToDomainTickBatch_Valid(t *testing.T) {
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
}

func TestToDomainTickBatch_PreRollAndBoundary(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	pb := &v1.TickBatch{
		TickIndex: 0,
		StartTime: timestamppb.New(now),
		EndTime:   timestamppb.New(now.Add(time.Second)),
		Events: []*v1.Event{
			{
				Payload: &v1.Event_Window{
					Window: &v1.WindowEvent{
						Timestamp:   timestamppb.New(now.Add(-500 * time.Millisecond)),
						Action:      "focus",
						WindowTitle: "Pre-roll Window",
						ProcessName: "init.exe",
					},
				},
			},
			{
				Payload: &v1.Event_Mouse{
					Mouse: &v1.MouseEvent{
						Timestamp: timestamppb.New(now.Add(time.Second)), // exactly on upper boundary
						Action:    "click",
						Button:    "left",
					},
				},
			},
		},
	}

	domainBatch := ToDomainTickBatch(pb)
	if domainBatch.TickIndex != 0 {
		t.Errorf("expected tickIndex 0, got %d", domainBatch.TickIndex)
	}
	if domainBatch.Len() != 2 {
		t.Fatalf("expected 2 events, got %d", domainBatch.Len())
	}
	evs := domainBatch.Events()
	if winEv, ok := evs[0].(events.WindowEvent); !ok || winEv.WindowTitle != "Pre-roll Window" {
		t.Errorf("unexpected first event: %+v", evs[0])
	}
	if mouseEv, ok := evs[1].(events.MouseEvent); !ok || mouseEv.Action != "click" {
		t.Errorf("unexpected second event: %+v", evs[1])
	}
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
	if !ok || mEv.Action != "click" || mEv.Position.X != 10 || mEv.Position.Y != 20 || !mEv.IsClick {
		t.Errorf("unexpected mouse event: %+v", ev)
	}
	if nilMouse := ToDomainMouseEvent(nil); nilMouse.Action != "" {
		t.Errorf("expected empty mouse event for nil")
	}
}

func TestIsClickEvent(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		button     string
		clickCount string
		expected   bool
	}{
		// Primary button clicks
		{name: "standard left click", action: "click", button: "left", expected: true},
		{name: "primary button click", action: "click", button: "primary", expected: true},
		{name: "main button click", action: "click", button: "main", expected: true},
		{name: "mousedown action", action: "mousedown", button: "primary", expected: true},
		{name: "mouse_click action", action: "mouse_click", button: "primary", expected: true},
		{name: "left button without action", button: "left", expected: true},
		{name: "primary button without action", button: "primary", expected: true},
		{name: "main button without action", button: "main", expected: true},
		{name: "click action without button", action: "click", expected: true},
		{name: "mousedown action without button", action: "mousedown", expected: true},
		{name: "mouse_click action without button", action: "mouse_click", expected: true},
		{name: "click count single", clickCount: "single", expected: true},
		{name: "click count double", clickCount: "double", expected: true},
		{name: "click count 1", clickCount: "1", expected: true},
		{name: "case insensitive primary button", action: "CLICK", button: "PRIMARY", expected: true},

		// Move & gesture rejections
		{name: "move action rejected", action: "move", button: "primary", expected: false},
		{name: "mousemove action rejected", action: "mousemove", button: "left", expected: false},
		{name: "drag action rejected", action: "drag", button: "left", expected: false},
		{name: "mousedrag action rejected", action: "mousedrag", button: "primary", expected: false},
		{name: "scroll action rejected", action: "scroll", button: "primary", expected: false},
		{name: "mousescroll action rejected", action: "mousescroll", button: "left", expected: false},
		{name: "up action rejected", action: "up", button: "left", expected: false},
		{name: "mouseup action rejected", action: "mouseup", button: "primary", expected: false},
		{name: "release action rejected", action: "release", button: "main", expected: false},
		{name: "move action with click count rejected", action: "move", clickCount: "single", expected: false},
		{name: "drag action with primary button rejected", action: "drag", button: "primary", clickCount: "1", expected: false},

		// Middle click rejections
		{name: "middle click rejected", action: "click", button: "middle", expected: false},
		{name: "middle mousedown rejected", action: "mousedown", button: "middle", expected: false},
		{name: "middle mouse_click rejected", action: "mouse_click", button: "middle", expected: false},
		{name: "middle button without action rejected", button: "middle", expected: false},

		// Right click rejections
		{name: "right click rejected", action: "click", button: "right", expected: false},
		{name: "right mousedown rejected", action: "mousedown", button: "right", expected: false},
		{name: "right mouse_click rejected", action: "mouse_click", button: "right", expected: false},
		{name: "right button without action rejected", button: "right", expected: false},

		// Click count zero / none rejections
		{name: "empty event rejected", expected: false},
		{name: "zero click count rejected", clickCount: "0", expected: false},
		{name: "none click count rejected", clickCount: "none", expected: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual := IsClickEvent(tc.action, tc.button, tc.clickCount)
			if actual != tc.expected {
				t.Errorf("IsClickEvent(%q, %q, %q) = %v, want %v", tc.action, tc.button, tc.clickCount, actual, tc.expected)
			}
		})
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
