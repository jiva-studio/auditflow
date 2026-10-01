// Package events defines domain event types captured from user desktop activity.
package events

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/geometry"
)

// Sentinel errors for event validation and tick batches.
var (
	ErrInvalidTickIndex = errors.New("tick index must be non-negative")
	ErrInvalidInterval  = errors.New("start time must be before end time and non-zero")
	ErrNilEvent         = errors.New("cannot add nil event to tick batch")
	ErrEventOutOfBounds = errors.New("event timestamp is outside tick interval")
)

// EventType identifies the classification of an event.
type EventType string

// Supported domain event types.
const (
	EventTypeWindow          EventType = "window"
	EventTypeMouse           EventType = "mouse"
	EventTypeClipboard       EventType = "clipboard"
	EventTypeOCR             EventType = "ocr"
	EventTypeKeyboard        EventType = "keyboard"
	EventTypeKeystroke       EventType = "keystroke"
	EventTypeMouseDrag       EventType = "mouse_drag"
	EventTypeMouseScroll     EventType = "mouse_scroll"
	EventTypeDisplayTopology EventType = "display_topology"
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

// DisplayTopologyEvent communicates the physical/virtual display layout of the workstation.
type DisplayTopologyEvent struct {
	Timestamp time.Time
	Displays  []display.Display
}

// GetTimestamp returns the event timestamp.
func (e DisplayTopologyEvent) GetTimestamp() time.Time { return e.Timestamp }

// GetType returns EventTypeDisplayTopology.
func (e DisplayTopologyEvent) GetType() EventType { return EventTypeDisplayTopology }

// TickBatch represents a batch of events occurring within a 1-second interval.
type TickBatch struct {
	TickIndex int
	StartTime time.Time
	EndTime   time.Time
	events    []Event
}

// NewTickBatch constructs and validates a new TickBatch with the given index and time interval.
func NewTickBatch(tickIndex int, startTime, endTime time.Time) (TickBatch, error) {
	if tickIndex < 0 {
		return TickBatch{}, fmt.Errorf("tick index %d: %w", tickIndex, ErrInvalidTickIndex)
	}
	if startTime.IsZero() || endTime.IsZero() || !startTime.Before(endTime) {
		return TickBatch{}, fmt.Errorf("interval [%v, %v): %w", startTime, endTime, ErrInvalidInterval)
	}
	return TickBatch{
		TickIndex: tickIndex,
		StartTime: startTime,
		EndTime:   endTime,
		events:    make([]Event, 0),
	}, nil
}

// Add appends an event to the batch while maintaining strict chronological ordering and time boundaries.
func (b *TickBatch) Add(event Event) error {
	if event == nil {
		return ErrNilEvent
	}
	ts := event.GetTimestamp()
	if b.TickIndex == 0 {
		if !ts.Before(b.EndTime) {
			return fmt.Errorf("event timestamp %v outside [-, %v): %w", ts, b.EndTime, ErrEventOutOfBounds)
		}
	} else {
		if ts.Before(b.StartTime) || !ts.Before(b.EndTime) {
			return fmt.Errorf("event timestamp %v outside [%v, %v): %w", ts, b.StartTime, b.EndTime, ErrEventOutOfBounds)
		}
	}

	idx := sort.Search(len(b.events), func(i int) bool {
		return !b.events[i].GetTimestamp().Before(ts)
	})
	b.events = append(b.events, nil)
	copy(b.events[idx+1:], b.events[idx:])
	b.events[idx] = event
	return nil
}

// Events returns a copy of the ordered events contained within the batch.
func (b TickBatch) Events() []Event {
	result := make([]Event, len(b.events))
	copy(result, b.events)
	return result
}

// Len returns the number of events in the batch.
func (b TickBatch) Len() int {
	return len(b.events)
}

// IsEmpty returns true if the batch contains no events.
func (b TickBatch) IsEmpty() bool {
	return len(b.events) == 0
}
