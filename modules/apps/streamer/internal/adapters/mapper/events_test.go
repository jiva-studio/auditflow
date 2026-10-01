package mapper_test

import (
	"testing"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/mapper"
	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/geometry"
)

type customUnsupportedEvent struct {
	ts time.Time
}

func (c customUnsupportedEvent) GetTimestamp() time.Time   { return c.ts }
func (c customUnsupportedEvent) GetType() events.EventType { return events.EventType("custom") }

var (
	sampleTime = time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	sampleRect = geometry.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080}
)

func TestToProtoWindowEvent(t *testing.T) {
	we := events.WindowEvent{
		Timestamp:       sampleTime,
		Action:          "focus_change",
		WindowTitle:     "Outlook",
		ProcessName:     "OUTLOOK.EXE",
		ApplicationName: "Outlook",
		WindowRect:      sampleRect,
		WindowState:     "normal",
		URL:             "https://example.com",
		Domain:          "example.com",
		DwellTimeMS:     1500,
	}
	pb := mapper.ToProtoEvent(we)
	if pb.GetWindow() == nil {
		t.Fatalf("expected WindowEvent payload")
	}
	if pb.GetWindow().WindowTitle != "Outlook" {
		t.Errorf("got %s, want Outlook", pb.GetWindow().WindowTitle)
	}
}

func TestToProtoMouseEvent(t *testing.T) {
	me := events.MouseEvent{
		Timestamp:   sampleTime,
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 100, Y: 200},
		ClickCount:  "single",
		WindowTitle: "Outlook",
		ProcessName: "OUTLOOK.EXE",
	}
	pb := mapper.ToProtoEvent(me)
	if pb.GetMouse() == nil {
		t.Fatalf("expected MouseEvent payload")
	}
	if pb.GetMouse().Position.X != 100 {
		t.Errorf("got x %d, want 100", pb.GetMouse().Position.X)
	}
}

func TestToProtoClipboardEvent(t *testing.T) {
	ce := events.ClipboardEvent{
		Timestamp: sampleTime,
		Action:    "clipboard_change",
		Text:      "INV-999",
		Length:    7,
		SourceApp: "OUTLOOK.EXE",
		Formats:   []string{"text/plain"},
	}
	pb := mapper.ToProtoEvent(ce)
	if pb.GetClipboard() == nil {
		t.Fatalf("expected ClipboardEvent payload")
	}
	if pb.GetClipboard().Text != "INV-999" {
		t.Errorf("got text %s, want INV-999", pb.GetClipboard().Text)
	}
}

func TestToProtoOCREvent(t *testing.T) {
	block := display.OCRTextBlock{
		Text:       "Invoice",
		Box:        geometry.Rectangle{X: 10, Y: 20, Width: 30, Height: 40},
		Confidence: 0.98,
	}
	oe := events.OCREvent{
		Timestamp:        sampleTime,
		Filename:         "screen.jpg",
		DisplayID:        1,
		Resolution:       geometry.Size{Width: 1920, Height: 1080},
		ScaleFactor:      1.5,
		WindowRect:       sampleRect,
		Blocks:           []display.OCRTextBlock{block},
		DeduplicatedFrom: "prev.jpg",
	}
	pb := mapper.ToProtoEvent(oe)
	if pb.GetOcr() == nil {
		t.Fatalf("expected OCREvent payload")
	}
	if len(pb.GetOcr().Blocks) != 1 || pb.GetOcr().Blocks[0].Text != "Invoice" {
		t.Errorf("got blocks %v, want 1 block Invoice", pb.GetOcr().Blocks)
	}
}

func TestToProtoGenericEvent(t *testing.T) {
	ge := events.GenericActivityEvent{
		Timestamp: sampleTime,
		Type:      events.EventTypeKeyboard,
	}
	pb := mapper.ToProtoEvent(ge)
	if pb.GetGeneric() == nil {
		t.Fatalf("expected GenericActivityEvent payload")
	}
	if pb.GetGeneric().Type != "keyboard" {
		t.Errorf("got type %s, want keyboard", pb.GetGeneric().Type)
	}
}

func TestToProtoUnsupportedEvent(t *testing.T) {
	ue := customUnsupportedEvent{ts: sampleTime}
	pb := mapper.ToProtoEvent(ue)
	if pb != nil {
		t.Errorf("expected nil for unsupported event, got %v", pb)
	}
}
