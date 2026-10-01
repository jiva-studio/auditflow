// Package desktop provides domain state tracking and spatial hit-testing for user desktops.
package desktop

import (
	"sort"

	"assessment/libs/domain/display"
	"assessment/libs/domain/events"
	"assessment/libs/domain/geometry"
	"assessment/libs/domain/rules"
)

// WindowFacet tracks the focused window state and process context.
type WindowFacet struct {
	activeWindow events.WindowEvent
}

// Apply updates the window facet with window events.
func (f *WindowFacet) Apply(event events.Event) {
	if ev, ok := event.(events.WindowEvent); ok {
		f.activeWindow = ev
	}
}

// ActiveWindow returns the current active window event.
func (f WindowFacet) ActiveWindow() events.WindowEvent {
	return f.activeWindow
}

// ProcessName returns the process name of the focused window.
func (f WindowFacet) ProcessName() string {
	return f.activeWindow.ProcessName
}

// WindowTitle returns the window title of the focused window.
func (f WindowFacet) WindowTitle() string {
	return f.activeWindow.WindowTitle
}

// ClipboardFacet tracks the active clipboard contents.
type ClipboardFacet struct {
	activeClipboard events.ClipboardEvent
}

// Apply updates the clipboard facet with clipboard events.
func (f *ClipboardFacet) Apply(event events.Event) {
	if ev, ok := event.(events.ClipboardEvent); ok {
		f.activeClipboard = ev
	}
}

// ActiveClipboard returns the current clipboard event.
func (f ClipboardFacet) ActiveClipboard() events.ClipboardEvent {
	return f.activeClipboard
}

// Text returns the text content of the active clipboard.
func (f ClipboardFacet) Text() string {
	return f.activeClipboard.Text
}

// SpatialFacet tracks multi-display topology and spatial visual layers (OCR, etc.).
type SpatialFacet struct {
	displays map[int]display.Display
	layers   map[int]display.SpatialLayer
}

// NewSpatialFacet initializes an empty SpatialFacet.
func NewSpatialFacet(displays ...display.Display) SpatialFacet {
	f := SpatialFacet{
		displays: make(map[int]display.Display, len(displays)),
		layers:   make(map[int]display.SpatialLayer),
	}
	f.SetDisplays(displays)
	return f
}

// SetDisplays updates the configured display topology.
func (f *SpatialFacet) SetDisplays(displays []display.Display) {
	f.displays = make(map[int]display.Display, len(displays))
	for _, d := range displays {
		f.displays[d.ID] = d
	}
}

// SetLayer registers or updates a spatial layer on a specific display.
func (f *SpatialFacet) SetLayer(displayID int, layer display.SpatialLayer) {
	if f.layers == nil {
		f.layers = make(map[int]display.SpatialLayer)
	}
	f.layers[displayID] = layer
}

// Apply updates spatial layers from domain events.
func (f *SpatialFacet) Apply(event events.Event) {
	ev, ok := event.(events.OCREvent)
	if !ok {
		return
	}

	scale := ev.ScaleFactor
	if scale <= 0 {
		scale = 1.0
	}
	f.autoRegisterDisplay(ev, scale)

	blocks := ev.Blocks
	if len(blocks) == 0 && ev.DeduplicatedFrom != "" {
		if prevLayer, has := f.layers[ev.DisplayID]; has {
			if prevFrame, ok := prevLayer.(display.OCRFrame); ok {
				blocks = prevFrame.Blocks
			}
		}
	}

	frame := display.OCRFrame{
		DisplayID:   ev.DisplayID,
		Resolution:  ev.Resolution,
		ScaleFactor: scale,
		Blocks:      blocks,
		Timestamp:   ev.Timestamp,
		Filename:    ev.Filename,
	}
	f.SetLayer(ev.DisplayID, frame)
}

func (f *SpatialFacet) autoRegisterDisplay(ev events.OCREvent, scale float64) {
	if ev.Resolution.Width <= 0 || ev.Resolution.Height <= 0 {
		return
	}
	if _, hasDisplay := f.displays[ev.DisplayID]; hasDisplay {
		return
	}

	var origX, origY int
	if ev.DisplayID != 0 {
		origX = ev.WindowRect.X
		origY = ev.WindowRect.Y
	}

	displayBounds := geometry.Rectangle{
		X:      origX,
		Y:      origY,
		Width:  ev.Resolution.Width,
		Height: ev.Resolution.Height,
	}
	disp, err := display.NewDisplay(ev.DisplayID, displayBounds, scale, ev.DisplayID == 0)
	if err != nil {
		return
	}
	if f.displays == nil {
		f.displays = make(map[int]display.Display)
	}
	f.displays[ev.DisplayID] = disp
}

// FindTextAt performs spatial hit-testing across registered displays and visual layers.
func (f *SpatialFacet) FindTextAt(point geometry.Point) (string, bool) {
	targetDisplay, found := f.findDisplayForPoint(point)
	if !found {
		return f.findInAllLayers(point)
	}

	localPt, ok := targetDisplay.MapToLocal(point)
	if !ok {
		return "", false
	}

	layer, hasLayer := f.layers[targetDisplay.ID]
	if !hasLayer {
		return "", false
	}

	hit, hitFound := layer.HitTest(localPt, targetDisplay.Bounds)
	if !hitFound {
		return "", false
	}
	return hit.Text, true
}

func (f *SpatialFacet) sortedDisplayIDs() []int {
	ids := make([]int, 0, len(f.displays))
	for id := range f.displays {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (f *SpatialFacet) sortedLayerIDs() []int {
	ids := make([]int, 0, len(f.layers))
	for id := range f.layers {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (f *SpatialFacet) findDisplayForPoint(point geometry.Point) (display.Display, bool) {
	for _, id := range f.sortedDisplayIDs() {
		d := f.displays[id]
		if d.Contains(point) {
			return d, true
		}
	}
	return display.Display{}, false
}

func (f *SpatialFacet) findInAllLayers(point geometry.Point) (string, bool) {
	for _, displayID := range f.sortedLayerIDs() {
		layer := f.layers[displayID]
		d, hasDisplay := f.displays[displayID]

		var bounds geometry.Rectangle
		if hasDisplay {
			bounds = d.Bounds
		} else if frame, ok := layer.(display.OCRFrame); ok && frame.Resolution.Width > 0 && frame.Resolution.Height > 0 {
			bounds = geometry.Rectangle{
				X:      0,
				Y:      0,
				Width:  frame.Resolution.Width,
				Height: frame.Resolution.Height,
			}
		} else {
			continue
		}

		if hit, hitFound := layer.HitTest(point, bounds); hitFound {
			return hit.Text, true
		}
	}
	return "", false
}

// AllTexts returns all visible text strings from all spatial layers in deterministic order.
func (f *SpatialFacet) AllTexts() []string {
	var texts []string
	for _, id := range f.sortedLayerIDs() {
		texts = append(texts, f.layers[id].AllTexts()...)
	}
	return texts
}

// State is the aggregate root maintaining the composite desktop environment.
type State struct {
	window    WindowFacet
	clipboard ClipboardFacet
	spatial   SpatialFacet
}

// NewState initializes a new State instance with optional displays.
func NewState(displays ...display.Display) *State {
	return &State{
		spatial: NewSpatialFacet(displays...),
	}
}

// SetDisplays updates the configured display topology.
func (s *State) SetDisplays(displays []display.Display) {
	s.spatial.SetDisplays(displays)
}

// Apply evolves the desktop state using the reducer pattern across all facets.
func (s *State) Apply(event events.Event) {
	if event == nil {
		return
	}
	s.window.Apply(event)
	s.clipboard.Apply(event)
	s.spatial.Apply(event)
}

// ActiveWindow returns the current active window event.
func (s *State) ActiveWindow() events.WindowEvent {
	return s.window.ActiveWindow()
}

// ActiveProcess returns the process name of the focused window.
func (s *State) ActiveProcess() string {
	return s.window.ProcessName()
}

// ActiveWindowTitle returns the window title of the focused window.
func (s *State) ActiveWindowTitle() string {
	return s.window.WindowTitle()
}

// ActiveClipboardText returns the current clipboard text.
func (s *State) ActiveClipboardText() string {
	return s.clipboard.Text()
}

// FindTextAt performs spatial hit-testing to locate the text under the given coordinates.
func (s *State) FindTextAt(point geometry.Point) (string, bool) {
	return s.spatial.FindTextAt(point)
}

// AllScreenTexts returns a slice of all visible text strings across all visual layers.
func (s *State) AllScreenTexts() []string {
	return s.spatial.AllTexts()
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
