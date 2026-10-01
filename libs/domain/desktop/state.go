// Package desktop provides domain state tracking and spatial hit-testing for user desktops.
package desktop

import (
	"assessment/libs/domain/display"
	"assessment/libs/domain/events"
	"assessment/libs/domain/geometry"
	"assessment/libs/domain/rules"
)

// State maintains the runtime snapshot of the desktop environment.
type State struct {
	displays        map[int]display.Display
	activeWindow    events.WindowEvent
	activeClipboard events.ClipboardEvent
	ocrFrames       map[int]display.OCRFrame
}

// NewState initializes a new State instance with optional displays.
func NewState(displays ...display.Display) *State {
	dispMap := make(map[int]display.Display, len(displays))
	for _, d := range displays {
		dispMap[d.ID] = d
	}
	return &State{
		displays:  dispMap,
		ocrFrames: make(map[int]display.OCRFrame),
	}
}

// SetDisplays updates the configured display topology.
func (s *State) SetDisplays(displays []display.Display) {
	s.displays = make(map[int]display.Display, len(displays))
	for _, d := range displays {
		s.displays[d.ID] = d
	}
}

// Apply updates the desktop state with a domain event.
func (s *State) Apply(event events.Event) {
	if event == nil {
		return
	}

	switch ev := event.(type) {
	case events.WindowEvent:
		s.activeWindow = ev
	case events.ClipboardEvent:
		s.activeClipboard = ev
	case events.OCREvent:
		s.applyOCREvent(ev)
	}
}

func (s *State) applyOCREvent(ev events.OCREvent) {
	s.ocrFrames[ev.DisplayID] = display.OCRFrame{
		DisplayID:   ev.DisplayID,
		Resolution:  ev.Resolution,
		ScaleFactor: ev.ScaleFactor,
		Blocks:      ev.Blocks,
		Timestamp:   ev.Timestamp,
		Filename:    ev.Filename,
	}
}

// ActiveWindow returns the current active window event.
func (s *State) ActiveWindow() events.WindowEvent {
	return s.activeWindow
}

// ActiveProcess returns the process name of the focused window.
func (s *State) ActiveProcess() string {
	return s.activeWindow.ProcessName
}

// ActiveWindowTitle returns the window title of the focused window.
func (s *State) ActiveWindowTitle() string {
	return s.activeWindow.WindowTitle
}

// ActiveClipboardText returns the current clipboard text.
func (s *State) ActiveClipboardText() string {
	return s.activeClipboard.Text
}

// FindTextAt performs spatial hit-testing to locate the OCR text block under the given coordinates.
func (s *State) FindTextAt(point geometry.Point) (string, bool) {
	targetDisplay, found := s.findDisplayForPoint(point)
	if !found {
		return s.findBlockInAllFrames(point)
	}

	localPt, ok := targetDisplay.MapToLocal(point)
	if !ok {
		return "", false
	}

	frame, hasFrame := s.ocrFrames[targetDisplay.ID]
	if !hasFrame {
		return "", false
	}

	block, hit := frame.FindBlockAt(localPt, targetDisplay.Bounds)
	if !hit {
		return "", false
	}
	return block.Text, true
}

func (s *State) findDisplayForPoint(point geometry.Point) (display.Display, bool) {
	for _, d := range s.displays {
		if d.Contains(point) {
			return d, true
		}
	}
	return display.Display{}, false
}

func (s *State) findBlockInAllFrames(point geometry.Point) (string, bool) {
	for _, frame := range s.ocrFrames {
		d, hasDisplay := s.displays[frame.DisplayID]
		bounds := geometry.Rectangle{
			X:      0,
			Y:      0,
			Width:  frame.Resolution.Width,
			Height: frame.Resolution.Height,
		}
		if hasDisplay {
			bounds = d.Bounds
		}

		if block, hit := frame.FindBlockAt(point, bounds); hit {
			return block.Text, true
		}
	}
	return "", false
}

// AllScreenTexts returns a slice of all visible OCR text block strings across all displays.
func (s *State) AllScreenTexts() []string {
	var texts []string
	for _, frame := range s.ocrFrames {
		for _, block := range frame.Blocks {
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		}
	}
	return texts
}

// BuildEvaluationContext creates a RuleEvaluationContext from current desktop state and click text.
func (s *State) BuildEvaluationContext(clickText string) rules.RuleEvaluationContext {
	return rules.RuleEvaluationContext{
		ClickText:      clickText,
		Process:        s.ActiveProcess(),
		WindowTitle:    s.ActiveWindowTitle(),
		ClipboardText:  s.ActiveClipboardText(),
		OCRScreenTexts: s.AllScreenTexts(),
	}
}
