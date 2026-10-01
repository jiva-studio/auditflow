// Package mapper converts domain models to protocol buffer representations.
package mapper

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/events"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

// ToProtoEvent converts a single domain Event into its corresponding Protobuf oneof wrapper.
func ToProtoEvent(ev events.Event) *v1.Event {
	switch e := ev.(type) {
	case events.WindowEvent:
		return ToProtoWindowEvent(e)
	case events.MouseEvent:
		return ToProtoMouseEvent(e)
	case events.ClipboardEvent:
		return ToProtoClipboardEvent(e)
	case events.OCREvent:
		return ToProtoOCREvent(e)
	case events.GenericActivityEvent:
		return ToProtoGenericEvent(e)
	case events.DisplayTopologyEvent:
		return ToProtoDisplayTopologyEvent(e)
	default:
		return nil
	}
}

// ToProtoWindowEvent maps a WindowEvent domain entity to Protobuf.
func ToProtoWindowEvent(e events.WindowEvent) *v1.Event {
	return &v1.Event{
		Payload: &v1.Event_Window{
			Window: &v1.WindowEvent{
				Timestamp:       timestamppb.New(e.Timestamp),
				Action:          e.Action,
				WindowTitle:     e.WindowTitle,
				ProcessName:     e.ProcessName,
				ApplicationName: e.ApplicationName,
				WindowRect:      ToProtoRectangle(e.WindowRect),
				WindowState:     e.WindowState,
				Url:             e.URL,
				Domain:          e.Domain,
				DwellTimeMs:     int32(e.DwellTimeMS),
			},
		},
	}
}

// ToProtoMouseEvent maps a MouseEvent domain entity to Protobuf.
func ToProtoMouseEvent(e events.MouseEvent) *v1.Event {
	return &v1.Event{
		Payload: &v1.Event_Mouse{
			Mouse: &v1.MouseEvent{
				Timestamp:   timestamppb.New(e.Timestamp),
				Action:      e.Action,
				Button:      e.Button,
				Position:    ToProtoPoint(e.Position),
				ClickCount:  e.ClickCount,
				WindowTitle: e.WindowTitle,
				ProcessName: e.ProcessName,
			},
		},
	}
}

// ToProtoClipboardEvent maps a ClipboardEvent domain entity to Protobuf.
func ToProtoClipboardEvent(e events.ClipboardEvent) *v1.Event {
	return &v1.Event{
		Payload: &v1.Event_Clipboard{
			Clipboard: &v1.ClipboardEvent{
				Timestamp: timestamppb.New(e.Timestamp),
				Action:    e.Action,
				Text:      e.Text,
				Length:    int32(e.Length),
				SourceApp: e.SourceApp,
				Formats:   e.Formats,
			},
		},
	}
}

// ToProtoOCREvent maps an OCREvent domain entity to Protobuf.
func ToProtoOCREvent(e events.OCREvent) *v1.Event {
	blocks := make([]*v1.OCRTextBlock, len(e.Blocks))
	for i, b := range e.Blocks {
		blocks[i] = ToProtoOCRTextBlock(b)
	}
	return &v1.Event{
		Payload: &v1.Event_Ocr{
			Ocr: &v1.OCREvent{
				Timestamp:        timestamppb.New(e.Timestamp),
				Filename:         e.Filename,
				DisplayId:        int32(e.DisplayID),
				Resolution:       ToProtoSize(e.Resolution),
				ScaleFactor:      e.ScaleFactor,
				WindowRect:       ToProtoRectangle(e.WindowRect),
				Blocks:           blocks,
				DeduplicatedFrom: e.DeduplicatedFrom,
			},
		},
	}
}

// ToProtoOCRTextBlock maps an OCRTextBlock value object to Protobuf.
func ToProtoOCRTextBlock(b display.OCRTextBlock) *v1.OCRTextBlock {
	return &v1.OCRTextBlock{
		Text:       b.Text,
		Box:        ToProtoRectangle(b.Box),
		Confidence: b.Confidence,
	}
}

// ToProtoGenericEvent maps a GenericActivityEvent domain entity to Protobuf.
func ToProtoGenericEvent(e events.GenericActivityEvent) *v1.Event {
	return &v1.Event{
		Payload: &v1.Event_Generic{
			Generic: &v1.GenericActivityEvent{
				Timestamp: timestamppb.New(e.Timestamp),
				Type:      string(e.Type),
			},
		},
	}
}

// ToProtoDisplayTopologyEvent maps a DisplayTopologyEvent domain entity to Protobuf.
func ToProtoDisplayTopologyEvent(e events.DisplayTopologyEvent) *v1.Event {
	displays := make([]*v1.Display, len(e.Displays))
	for i, d := range e.Displays {
		displays[i] = ToProtoDisplay(d)
	}
	return &v1.Event{
		Payload: &v1.Event_DisplayTopology{
			DisplayTopology: &v1.DisplayTopologyEvent{
				Timestamp: timestamppb.New(e.Timestamp),
				Displays:  displays,
			},
		},
	}
}

// ToProtoDisplay maps a Display value object to Protobuf.
func ToProtoDisplay(d display.Display) *v1.Display {
	return &v1.Display{
		Id:      int32(d.ID),
		Bounds:  ToProtoRectangle(d.Bounds),
		Scale:   d.Scale,
		Primary: d.Primary,
	}
}
