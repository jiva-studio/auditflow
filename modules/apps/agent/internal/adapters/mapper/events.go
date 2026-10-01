package mapper

import (
	"time"

	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/events"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

// ToDomainTickBatch converts a Protobuf TickBatch message into a domain TickBatch aggregate.
func ToDomainTickBatch(pb *v1.TickBatch) events.TickBatch {
	if pb == nil {
		return events.TickBatch{}
	}

	startTime := pb.GetStartTime().AsTime()
	endTime := pb.GetEndTime().AsTime()

	batch, err := events.NewTickBatch(int(pb.GetTickIndex()), startTime, endTime)
	if err != nil {
		return events.TickBatch{}
	}

	for _, rawEv := range pb.GetEvents() {
		if ev := ToDomainEvent(rawEv); ev != nil {
			ts := ev.GetTimestamp()
			if pb.GetTickIndex() > 0 && ts.Before(batch.StartTime) {
				batch.StartTime = ts
			}
			if !ts.Before(batch.EndTime) {
				batch.EndTime = ts.Add(time.Nanosecond)
			}
			_ = batch.Add(ev)
		}
	}

	return batch
}

// ToDomainEvent converts a single Protobuf Event wrapper into its domain Event equivalent.
func ToDomainEvent(raw *v1.Event) events.Event {
	if raw == nil || raw.Payload == nil {
		return nil
	}

	switch p := raw.Payload.(type) {
	case *v1.Event_Window:
		return ToDomainWindowEvent(p.Window)
	case *v1.Event_Mouse:
		return ToDomainMouseEvent(p.Mouse)
	case *v1.Event_Clipboard:
		return ToDomainClipboardEvent(p.Clipboard)
	case *v1.Event_Ocr:
		return ToDomainOCREvent(p.Ocr)
	case *v1.Event_Generic:
		return ToDomainGenericEvent(p.Generic)
	case *v1.Event_DisplayTopology:
		return ToDomainDisplayTopologyEvent(p.DisplayTopology)
	default:
		return nil
	}
}

// ToDomainWindowEvent maps a Protobuf WindowEvent to domain WindowEvent.
func ToDomainWindowEvent(w *v1.WindowEvent) events.WindowEvent {
	if w == nil {
		return events.WindowEvent{}
	}
	return events.WindowEvent{
		Timestamp:       w.GetTimestamp().AsTime(),
		Action:          w.GetAction(),
		WindowTitle:     w.GetWindowTitle(),
		ProcessName:     w.GetProcessName(),
		ApplicationName: w.GetApplicationName(),
		WindowRect:      ToDomainRectangle(w.GetWindowRect()),
		WindowState:     w.GetWindowState(),
		URL:             w.GetUrl(),
		Domain:          w.GetDomain(),
		DwellTimeMS:     int(w.GetDwellTimeMs()),
	}
}

// ToDomainMouseEvent maps a Protobuf MouseEvent to domain MouseEvent.
func ToDomainMouseEvent(m *v1.MouseEvent) events.MouseEvent {
	if m == nil {
		return events.MouseEvent{}
	}
	return events.MouseEvent{
		Timestamp:   m.GetTimestamp().AsTime(),
		Action:      m.GetAction(),
		Button:      m.GetButton(),
		Position:    ToDomainPoint(m.GetPosition()),
		ClickCount:  m.GetClickCount(),
		WindowTitle: m.GetWindowTitle(),
		ProcessName: m.GetProcessName(),
	}
}

// ToDomainClipboardEvent maps a Protobuf ClipboardEvent to domain ClipboardEvent.
func ToDomainClipboardEvent(c *v1.ClipboardEvent) events.ClipboardEvent {
	if c == nil {
		return events.ClipboardEvent{}
	}
	return events.ClipboardEvent{
		Timestamp: c.GetTimestamp().AsTime(),
		Action:    c.GetAction(),
		Text:      c.GetText(),
		Length:    int(c.GetLength()),
		SourceApp: c.GetSourceApp(),
		Formats:   c.GetFormats(),
	}
}

// ToDomainOCREvent maps a Protobuf OCREvent to domain OCREvent.
func ToDomainOCREvent(o *v1.OCREvent) events.OCREvent {
	if o == nil {
		return events.OCREvent{}
	}
	rawBlocks := o.GetBlocks()
	blocks := make([]display.OCRTextBlock, len(rawBlocks))
	for i, b := range rawBlocks {
		blocks[i] = ToDomainOCRTextBlock(b)
	}

	return events.OCREvent{
		Timestamp:        o.GetTimestamp().AsTime(),
		Filename:         o.GetFilename(),
		DisplayID:        int(o.GetDisplayId()),
		Resolution:       ToDomainSize(o.GetResolution()),
		ScaleFactor:      o.GetScaleFactor(),
		WindowRect:       ToDomainRectangle(o.GetWindowRect()),
		Blocks:           blocks,
		DeduplicatedFrom: o.GetDeduplicatedFrom(),
	}
}

// ToDomainGenericEvent maps a Protobuf GenericActivityEvent to domain GenericActivityEvent.
func ToDomainGenericEvent(g *v1.GenericActivityEvent) events.GenericActivityEvent {
	if g == nil {
		return events.GenericActivityEvent{}
	}
	return events.GenericActivityEvent{
		Timestamp: g.GetTimestamp().AsTime(),
		Type:      events.EventType(g.GetType()),
	}
}

// ToDomainDisplayTopologyEvent maps a Protobuf DisplayTopologyEvent to domain DisplayTopologyEvent.
func ToDomainDisplayTopologyEvent(d *v1.DisplayTopologyEvent) events.DisplayTopologyEvent {
	if d == nil {
		return events.DisplayTopologyEvent{}
	}
	rawDisplays := d.GetDisplays()
	displays := make([]display.Display, 0, len(rawDisplays))
	for _, rd := range rawDisplays {
		if disp, err := ToDomainDisplay(rd); err == nil {
			displays = append(displays, disp)
		}
	}
	return events.DisplayTopologyEvent{
		Timestamp: d.GetTimestamp().AsTime(),
		Displays:  displays,
	}
}

// ToDomainDisplay maps a Protobuf Display message to domain Display value object.
func ToDomainDisplay(d *v1.Display) (display.Display, error) {
	if d == nil {
		return display.Display{}, display.ErrInvalidDisplayBounds
	}
	bounds := ToDomainRectangle(d.GetBounds())
	return display.NewDisplay(int(d.GetId()), bounds, d.GetScale(), d.GetPrimary())
}
