// Package events defines domain event types captured from user desktop activity.
package events

import (
	"time"

	"assessment/libs/domain/display"
	"assessment/libs/domain/geometry"
)

// EventType identifies the classification of an event.
type EventType string

// Supported domain event types.
const (
	EventTypeWindow      EventType = "window"
	EventTypeMouse       EventType = "mouse"
	EventTypeClipboard   EventType = "clipboard"
	EventTypeOCR         EventType = "ocr"
	EventTypeKeyboard    EventType = "keyboard"
	EventTypeKeystroke   EventType = "keystroke"
	EventTypeMouseDrag   EventType = "mouse_drag"
	EventTypeMouseScroll EventType = "mouse_scroll"
)

// Event is the interface that all domain events implement.
type Event interface {
	GetTimestamp() time.Time
	GetType() EventType
}

// WindowEvent represents a window focus change event.
type WindowEvent struct {
	Timestamp       time.Time
	Action          string // "focus_change"
	WindowTitle     string
	ProcessName     string
	ApplicationName string
	WindowRect      geometry.Rectangle
	WindowState     string
	URL             string
	Domain          string
	DwellTimeMS     int
}

// GetTimestamp returns the event timestamp.
func (e WindowEvent) GetTimestamp() time.Time { return e.Timestamp }

// GetType returns EventTypeWindow.
func (e WindowEvent) GetType() EventType { return EventTypeWindow }

// MouseEvent represents a mouse click event.
type MouseEvent struct {
	Timestamp   time.Time
	Action      string // "click"
	Button      string // "left", "right"
	Position    geometry.Point
	ClickCount  string
	WindowTitle string
	ProcessName string
}

// GetTimestamp returns the event timestamp.
func (e MouseEvent) GetTimestamp() time.Time { return e.Timestamp }

// GetType returns EventTypeMouse.
func (e MouseEvent) GetType() EventType { return EventTypeMouse }

// ClipboardEvent represents a clipboard text change event.
type ClipboardEvent struct {
	Timestamp time.Time
	Action    string // "clipboard_change"
	Text      string
	Length    int
	SourceApp string
	Formats   []string
}

// GetTimestamp returns the event timestamp.
func (e ClipboardEvent) GetTimestamp() time.Time { return e.Timestamp }

// GetType returns EventTypeClipboard.
func (e ClipboardEvent) GetType() EventType { return EventTypeClipboard }

// OCREvent represents an OCR screen capture event.
type OCREvent struct {
	Timestamp        time.Time
	Filename         string
	DisplayID        int
	Resolution       geometry.Size
	ScaleFactor      float64
	WindowRect       geometry.Rectangle
	Blocks           []display.OCRTextBlock
	DeduplicatedFrom string
}

// GetTimestamp returns the event timestamp.
func (e OCREvent) GetTimestamp() time.Time { return e.Timestamp }

// GetType returns EventTypeOCR.
func (e OCREvent) GetType() EventType { return EventTypeOCR }

// GenericActivityEvent represents secondary user activity (keystroke, keyboard burst, scroll, drag).
type GenericActivityEvent struct {
	Timestamp time.Time
	Type      EventType
	Details   map[string]any
}

// GetTimestamp returns the event timestamp.
func (e GenericActivityEvent) GetTimestamp() time.Time { return e.Timestamp }

// GetType returns the generic event type.
func (e GenericActivityEvent) GetType() EventType { return e.Type }

// TickBatch represents a batch of events occurring within a 1-second interval.
type TickBatch struct {
	TickIndex int
	StartTime time.Time
	EndTime   time.Time
	Events    []Event
}
