package desktop_test

import (
	"testing"
	"time"

	"assessment/libs/domain/desktop"
	"assessment/libs/domain/display"
	"assessment/libs/domain/events"
	"assessment/libs/domain/geometry"
)

func createSampleDisplay(t *testing.T, id int, x, y, w, h int, scale float64, primary bool) display.Display {
	t.Helper()
	bounds, err := geometry.NewRectangle(x, y, w, h)
	if err != nil {
		t.Fatalf("failed to create bounds: %v", err)
	}
	d, err := display.NewDisplay(id, bounds, scale, primary)
	if err != nil {
		t.Fatalf("failed to create display: %v", err)
	}
	return d
}

func TestDesktopState_InitializationAndSetDisplays(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	if state.ActiveProcess() != "" {
		t.Errorf("expected empty active process, got %q", state.ActiveProcess())
	}
	if state.ActiveWindowTitle() != "" {
		t.Errorf("expected empty active window title, got %q", state.ActiveWindowTitle())
	}
	if state.ActiveClipboardText() != "" {
		t.Errorf("expected empty clipboard, got %q", state.ActiveClipboardText())
	}

	d1 := createSampleDisplay(t, 1, 1920, 0, 1920, 1080, 1.0, false)
	state.SetDisplays([]display.Display{d0, d1})

	// Check that state still functions cleanly
	if len(state.AllScreenTexts()) != 0 {
		t.Errorf("expected 0 screen texts, got %d", len(state.AllScreenTexts()))
	}
}

func TestDesktopState_ApplyEvents(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	// 1. Nil event should not panic
	state.Apply(nil)

	// 2. Window event
	now := time.Now()
	rect, _ := geometry.NewRectangle(100, 100, 800, 600)
	winEv := events.WindowEvent{
		Timestamp:       now,
		Action:          "focus_change",
		WindowTitle:     "Inbox - Outlook",
		ProcessName:     "OUTLOOK.EXE",
		ApplicationName: "Outlook",
		WindowRect:      rect,
	}
	state.Apply(winEv)

	if state.ActiveProcess() != "OUTLOOK.EXE" {
		t.Errorf("got process %q, want OUTLOOK.EXE", state.ActiveProcess())
	}
	if state.ActiveWindowTitle() != "Inbox - Outlook" {
		t.Errorf("got window title %q, want Inbox - Outlook", state.ActiveWindowTitle())
	}
	if state.ActiveWindow().ApplicationName != "Outlook" {
		t.Errorf("got app name %q, want Outlook", state.ActiveWindow().ApplicationName)
	}

	// 3. Clipboard event
	clipEv := events.ClipboardEvent{
		Timestamp: now,
		Action:    "clipboard_change",
		Text:      "INV-998877",
		Length:    10,
	}
	state.Apply(clipEv)

	if state.ActiveClipboardText() != "INV-998877" {
		t.Errorf("got clipboard %q, want INV-998877", state.ActiveClipboardText())
	}

	// 4. Generic event should not break state
	state.Apply(events.GenericActivityEvent{Timestamp: now, Type: events.EventTypeKeyboard})
}

func TestDesktopState_SpatialHitTesting_SingleDisplay(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	box1, _ := geometry.NewRectangle(100, 200, 300, 50)
	box2, _ := geometry.NewRectangle(500, 200, 200, 50)

	ocrEv := events.OCREvent{
		Timestamp:   time.Now(),
		DisplayID:   0,
		Resolution:  geometry.Size{Width: 1920, Height: 1080},
		ScaleFactor: 1.0,
		Blocks: []display.OCRTextBlock{
			{Text: "FW: Urgent Invoice", Box: box1, Confidence: 0.99},
			{Text: "Delete", Box: box2, Confidence: 0.95},
		},
	}
	state.Apply(ocrEv)

	tests := []struct {
		name     string
		point    geometry.Point
		wantText string
		wantHit  bool
	}{
		{"hit first block center", geometry.Point{X: 150, Y: 220}, "FW: Urgent Invoice", true},
		{"hit first block top-left", geometry.Point{X: 100, Y: 200}, "FW: Urgent Invoice", true},
		{"hit second block", geometry.Point{X: 550, Y: 220}, "Delete", true},
		{"miss between blocks", geometry.Point{X: 450, Y: 220}, "", false},
		{"miss above block", geometry.Point{X: 150, Y: 100}, "", false},
		{"miss out of display bounds", geometry.Point{X: 2000, Y: 200}, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, hit := state.FindTextAt(tc.point)
			if hit != tc.wantHit {
				t.Errorf("hit = %v, want %v", hit, tc.wantHit)
			}
			if text != tc.wantText {
				t.Errorf("text = %q, want %q", text, tc.wantText)
			}
		})
	}

	// Test display configured but without OCR frame yet
	dEmpty := createSampleDisplay(t, 99, 0, 0, 1000, 1000, 1.0, false)
	stateEmpty := desktop.NewState(dEmpty)
	if _, hit := stateEmpty.FindTextAt(geometry.Point{X: 100, Y: 100}); hit {
		t.Errorf("expected false when no frame is available for display")
	}
}

func TestDesktopState_SpatialHitTesting_MultiDisplay(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	d1 := createSampleDisplay(t, 1, 1920, 0, 1920, 1080, 1.0, false)
	state := desktop.NewState(d0, d1)

	boxDisp0, _ := geometry.NewRectangle(100, 100, 200, 50)
	boxDisp1, _ := geometry.NewRectangle(50, 50, 200, 50)

	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Display0 Text", Box: boxDisp0}},
	})
	state.Apply(events.OCREvent{
		DisplayID:  1,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Display1 Text", Box: boxDisp1}},
	})

	// Click on display 0
	text0, hit0 := state.FindTextAt(geometry.Point{X: 150, Y: 120})
	if !hit0 || text0 != "Display0 Text" {
		t.Errorf("display 0 hit-test failed: text=%q, hit=%v", text0, hit0)
	}

	// Click on display 1 (global coordinates: x = 1920 + 70 = 1990, y = 60)
	text1, hit1 := state.FindTextAt(geometry.Point{X: 1990, Y: 60})
	if !hit1 || text1 != "Display1 Text" {
		t.Errorf("display 1 hit-test failed: text=%q, hit=%v", text1, hit1)
	}

	// All screen texts
	allTexts := state.AllScreenTexts()
	if len(allTexts) != 2 {
		t.Errorf("got %d screen texts, want 2", len(allTexts))
	}
}

func TestDesktopState_FallbackWithoutConfiguredDisplays(t *testing.T) {
	// State without predefined displays
	state := desktop.NewState()

	box, _ := geometry.NewRectangle(100, 100, 200, 50)
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Fallback Text", Box: box}},
	})

	text, hit := state.FindTextAt(geometry.Point{X: 150, Y: 120})
	if !hit || text != "Fallback Text" {
		t.Errorf("fallback hit-test failed: got %q, %v", text, hit)
	}

	_, miss := state.FindTextAt(geometry.Point{X: 500, Y: 500})
	if miss {
		t.Errorf("expected miss, got hit")
	}
}

func TestDesktopState_BuildEvaluationContext(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	rect, _ := geometry.NewRectangle(0, 0, 800, 600)
	state.Apply(events.WindowEvent{
		WindowTitle: "Salesforce Account",
		ProcessName: "chrome.exe",
		WindowRect:  rect,
	})
	state.Apply(events.ClipboardEvent{
		Text: "ACCT-123",
	})
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{Text: "Are you sure you want to delete this record?"},
		},
	})

	ctx := state.BuildEvaluationContext("Delete")
	if ctx.ClickText != "Delete" {
		t.Errorf("got click %q, want Delete", ctx.ClickText)
	}
	if ctx.Process != "chrome.exe" {
		t.Errorf("got process %q, want chrome.exe", ctx.Process)
	}
	if ctx.WindowTitle != "Salesforce Account" {
		t.Errorf("got title %q, want Salesforce Account", ctx.WindowTitle)
	}
	if ctx.ClipboardText != "ACCT-123" {
		t.Errorf("got clipboard %q, want ACCT-123", ctx.ClipboardText)
	}
	if len(ctx.OCRScreenTexts) != 1 || ctx.OCRScreenTexts[0] != "Are you sure you want to delete this record?" {
		t.Errorf("got OCR texts %v", ctx.OCRScreenTexts)
	}
}

func TestDesktopState_SpatialHitTesting_NegativeDisplayCoordinates(t *testing.T) {
	// Monitor 0 is Left secondary monitor [-1920, 0, 1920, 1080]
	// Monitor 1 is Primary center monitor [0, 0, 1920, 1080]
	// Monitor 2 is Top secondary monitor [0, -1080, 1920, 1080]
	dLeft := createSampleDisplay(t, 0, -1920, 0, 1920, 1080, 1.0, false)
	dCenter := createSampleDisplay(t, 1, 0, 0, 1920, 1080, 1.0, true)
	dTop := createSampleDisplay(t, 2, 0, -1080, 1920, 1080, 1.0, false)

	state := desktop.NewState(dLeft, dCenter, dTop)

	boxLeft, _ := geometry.NewRectangle(100, 100, 200, 50)
	boxTop, _ := geometry.NewRectangle(200, 200, 300, 60)

	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Left Monitor Text", Box: boxLeft}},
	})
	state.Apply(events.OCREvent{
		DisplayID:  2,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Top Monitor Text", Box: boxTop}},
	})

	// Click in Left monitor: global coordinates X = -1920 + 150 = -1770, Y = 120
	textLeft, hitLeft := state.FindTextAt(geometry.Point{X: -1770, Y: 120})
	if !hitLeft || textLeft != "Left Monitor Text" {
		t.Errorf("left monitor hit-test failed: text=%q, hit=%v", textLeft, hitLeft)
	}

	// Click in Top monitor: global coordinates X = 250, Y = -1080 + 220 = -860
	textTop, hitTop := state.FindTextAt(geometry.Point{X: 250, Y: -860})
	if !hitTop || textTop != "Top Monitor Text" {
		t.Errorf("top monitor hit-test failed: text=%q, hit=%v", textTop, hitTop)
	}

	// Miss on Left monitor
	_, hitMiss := state.FindTextAt(geometry.Point{X: -1000, Y: 500})
	if hitMiss {
		t.Errorf("expected miss on left monitor empty space")
	}
}

func TestDesktopState_SpatialHitTesting_ResolutionScaling(t *testing.T) {
	// Virtual display bounds are 1920x1080, but OCR capture resolution is 3840x2160 (High-DPI 2x)
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 2.0, true)
	state := desktop.NewState(d0)

	// Block in 3840x2160 image coordinates at [200, 400, 600, 100]
	// Scaled down to virtual coordinates: [100, 200, 300, 50]
	boxOCR, _ := geometry.NewRectangle(200, 400, 600, 100)
	state.Apply(events.OCREvent{
		DisplayID:   0,
		Resolution:  geometry.Size{Width: 3840, Height: 2160},
		ScaleFactor: 2.0,
		Blocks:      []display.OCRTextBlock{{Text: "4K Scaled Invoice", Box: boxOCR}},
	})

	// Hit virtual point at (200, 220)
	text, hit := state.FindTextAt(geometry.Point{X: 200, Y: 220})
	if !hit || text != "4K Scaled Invoice" {
		t.Errorf("scaled hit-test failed: text=%q, hit=%v", text, hit)
	}

	// Miss virtual point outside scaled box (e.g. x = 450)
	_, miss := state.FindTextAt(geometry.Point{X: 450, Y: 220})
	if miss {
		t.Errorf("expected miss for point outside scaled OCR box")
	}
}

func TestDesktopState_SpatialHitTesting_ExactPixelBoundaries(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	// Block at [100, 100, 50, 50] -> X in [100, 150), Y in [100, 150)
	box, _ := geometry.NewRectangle(100, 100, 50, 50)
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Boundary Text", Box: box}},
	})

	boundaryCases := []struct {
		name    string
		point   geometry.Point
		wantHit bool
	}{
		{"top-left inside", geometry.Point{X: 100, Y: 100}, true},
		{"bottom-right inside", geometry.Point{X: 149, Y: 149}, true},
		{"top-right inside", geometry.Point{X: 149, Y: 100}, true},
		{"bottom-left inside", geometry.Point{X: 100, Y: 149}, true},
		{"right edge outside", geometry.Point{X: 150, Y: 120}, false},
		{"bottom edge outside", geometry.Point{X: 120, Y: 150}, false},
		{"left edge outside", geometry.Point{X: 99, Y: 120}, false},
		{"top edge outside", geometry.Point{X: 120, Y: 99}, false},
	}

	for _, bc := range boundaryCases {
		t.Run(bc.name, func(t *testing.T) {
			_, hit := state.FindTextAt(bc.point)
			if hit != bc.wantHit {
				t.Errorf("point %v: got hit=%v, want %v", bc.point, hit, bc.wantHit)
			}
		})
	}
}

func TestDesktopState_OCRFrameLifecycleReplacement(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	box, _ := geometry.NewRectangle(100, 100, 200, 50)

	// 1. Frame 1 with "Old Document"
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Old Document", Box: box}},
	})

	text1, hit1 := state.FindTextAt(geometry.Point{X: 150, Y: 120})
	if !hit1 || text1 != "Old Document" {
		t.Fatalf("frame 1 failed: got %q, %v", text1, hit1)
	}

	// 2. Frame 2 arrives for the same display with "New Document"
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "New Document", Box: box}},
	})

	text2, hit2 := state.FindTextAt(geometry.Point{X: 150, Y: 120})
	if !hit2 || text2 != "New Document" {
		t.Fatalf("frame 2 replacement failed: got %q, %v", text2, hit2)
	}

	// "Old Document" should not appear in AllScreenTexts
	texts := state.AllScreenTexts()
	if len(texts) != 1 || texts[0] != "New Document" {
		t.Errorf("expected only 'New Document' in texts, got %v", texts)
	}
}

func TestDesktopState_EmptyAndZeroBlocks(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	// 1. Empty blocks in OCR event
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{},
	})
	if len(state.AllScreenTexts()) != 0 {
		t.Errorf("expected 0 texts, got %d", len(state.AllScreenTexts()))
	}

	// 2. Block with empty text
	box, _ := geometry.NewRectangle(10, 10, 50, 50)
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "", Box: box}},
	})
	if len(state.AllScreenTexts()) != 0 {
		t.Errorf("expected empty text to be filtered out, got %v", state.AllScreenTexts())
	}

	// 3. Zero/invalid resolution OCR event
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 0, Height: 0},
		Blocks:     []display.OCRTextBlock{{Text: "Zero Resolution", Box: box}},
	})
	if _, hit := state.FindTextAt(geometry.Point{X: 20, Y: 20}); hit {
		t.Errorf("expected false for zero resolution frame")
	}
}

func TestDesktopState_OverlappingBlocks_Precedence(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	box1, _ := geometry.NewRectangle(100, 100, 200, 200)
	box2, _ := geometry.NewRectangle(150, 150, 100, 100)

	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{Text: "First Block", Box: box1},
			{Text: "Second Overlapping Block", Box: box2},
		},
	})

	// Click at (175, 175) which falls inside both box1 and box2
	text, hit := state.FindTextAt(geometry.Point{X: 175, Y: 175})
	if !hit || text != "First Block" {
		t.Errorf("expected first matching block, got text=%q, hit=%v", text, hit)
	}
}

func TestDesktopState_WindowAndClipboardTransitions(t *testing.T) {
	state := desktop.NewState()

	// Initial
	if state.ActiveProcess() != "" || state.ActiveClipboardText() != "" {
		t.Fatalf("expected empty initial state")
	}

	// Window 1
	state.Apply(events.WindowEvent{
		WindowTitle: "Document.docx - Word",
		ProcessName: "WINWORD.EXE",
	})
	if state.ActiveProcess() != "WINWORD.EXE" || state.ActiveWindowTitle() != "Document.docx - Word" {
		t.Errorf("window 1 transition failed")
	}

	// Window 2
	state.Apply(events.WindowEvent{
		WindowTitle: "Calculator",
		ProcessName: "calc.exe",
	})
	if state.ActiveProcess() != "calc.exe" || state.ActiveWindowTitle() != "Calculator" {
		t.Errorf("window 2 transition failed")
	}

	// Clipboard 1
	state.Apply(events.ClipboardEvent{Text: "Secret Password 123"})
	if state.ActiveClipboardText() != "Secret Password 123" {
		t.Errorf("clipboard 1 failed")
	}

	// Clipboard cleared to ""
	state.Apply(events.ClipboardEvent{Text: ""})
	if state.ActiveClipboardText() != "" {
		t.Errorf("clipboard clearing failed")
	}
}

func TestDesktopState_NegativeScenarios_Invariants(t *testing.T) {
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	state := desktop.NewState(d0)

	// 1. Extreme integer coordinates do not panic or false-positive hit
	extremePoints := []geometry.Point{
		{X: -9999999, Y: -9999999},
		{X: 9999999, Y: 9999999},
		{X: -1, Y: -1},
		{X: 1920, Y: 1080},
	}
	for _, p := range extremePoints {
		if text, hit := state.FindTextAt(p); hit {
			t.Errorf("expected miss for extreme point %v, got %q", p, text)
		}
	}

	// 2. SetDisplays(nil) cleans up displays, subsequent calls do not panic
	state.SetDisplays(nil)
	if text, hit := state.FindTextAt(geometry.Point{X: 100, Y: 100}); hit {
		t.Errorf("expected miss after clearing displays, got %q", text)
	}

	// 3. Negative displays hit-test with coordinates outside any display
	dLeft := createSampleDisplay(t, 1, -1920, 0, 1920, 1080, 1.0, false)
	state.SetDisplays([]display.Display{dLeft})
	if text, hit := state.FindTextAt(geometry.Point{X: -2000, Y: 500}); hit {
		t.Errorf("expected miss for point outside negative display, got %q", text)
	}

	// 4. Fallback search with OCR frame having zero resolution
	stateEmpty := desktop.NewState()
	stateEmpty.Apply(events.OCREvent{
		DisplayID:  5,
		Resolution: geometry.Size{Width: 0, Height: 0},
		Blocks: []display.OCRTextBlock{
			{Text: "Invalid", Box: geometry.Rectangle{X: 0, Y: 0, Width: 10, Height: 10}},
		},
	})
	if text, hit := stateEmpty.FindTextAt(geometry.Point{X: 5, Y: 5}); hit {
		t.Errorf("expected miss for fallback with zero resolution, got %q", text)
	}
}

func TestDesktopState_FacetsDirectAccess(t *testing.T) {
	// Test WindowFacet
	var winFacet desktop.WindowFacet
	winFacet.Apply(events.WindowEvent{ProcessName: "test.exe", WindowTitle: "Test Title"})
	if winFacet.ProcessName() != "test.exe" || winFacet.WindowTitle() != "Test Title" {
		t.Errorf("window facet getter failed")
	}

	// Test ClipboardFacet
	var clipFacet desktop.ClipboardFacet
	clipFacet.Apply(events.ClipboardEvent{Text: "Sample Clip", Length: 11})
	if clipFacet.Text() != "Sample Clip" || clipFacet.ActiveClipboard().Length != 11 {
		t.Errorf("clipboard facet getter failed")
	}

	// Test SpatialFacet custom layer registration
	spatialFacet := desktop.NewSpatialFacet()
	frame := display.OCRFrame{
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Custom Layer Text", Box: geometry.Rectangle{X: 0, Y: 0, Width: 100, Height: 100}}},
	}
	spatialFacet.SetLayer(0, frame)
	if text, hit := spatialFacet.FindTextAt(geometry.Point{X: 50, Y: 50}); !hit || text != "Custom Layer Text" {
		t.Errorf("custom layer hit-test failed: %q, %v", text, hit)
	}
}
