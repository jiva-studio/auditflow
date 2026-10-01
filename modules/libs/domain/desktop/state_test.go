package desktop_test

import (
	"testing"
	"time"

	"assessment/modules/libs/domain/desktop"
	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/geometry"
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

func TestDesktopState_FailsWithoutConfiguredTopology(t *testing.T) {
	// State without predefined displays
	state := desktop.NewState()

	box, _ := geometry.NewRectangle(100, 100, 200, 50)
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Fallback Text", Box: box}},
	})

	// Without display topology, OCR event for unknown display is ignored and hit-test fails
	text, hit := state.FindTextAt(geometry.Point{X: 150, Y: 120})
	if hit || text != "" {
		t.Errorf("expected hit test to fail without topology, got %q, %v", text, hit)
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
	d0 := createSampleDisplay(t, 0, 0, 0, 1920, 1080, 1.0, true)
	spatialFacet := desktop.NewSpatialFacet(d0)
	frame := display.OCRFrame{
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Custom Layer Text", Box: geometry.Rectangle{X: 0, Y: 0, Width: 100, Height: 100}}},
	}
	spatialFacet.SetLayer(0, frame)
	if text, hit := spatialFacet.FindTextAt(geometry.Point{X: 50, Y: 50}); !hit || text != "Custom Layer Text" {
		t.Errorf("custom layer hit-test failed: %q, %v", text, hit)
	}
}

func TestDesktopState_DeterministicMultiLayerIteration(t *testing.T) {
	d1 := createSampleDisplay(t, 1, 0, 0, 1920, 1080, 1.0, false)
	d2 := createSampleDisplay(t, 2, 0, 0, 1920, 1080, 1.0, false)
	d3 := createSampleDisplay(t, 3, 0, 0, 1920, 1080, 1.0, false)
	spatialFacet := desktop.NewSpatialFacet(d1, d2, d3)

	// Register multiple layers (ID 1, 2, 3) with overlapping bounding boxes
	box := geometry.Rectangle{X: 0, Y: 0, Width: 100, Height: 100}
	spatialFacet.SetLayer(3, display.OCRFrame{
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Layer 3 Text", Box: box}},
	})
	spatialFacet.SetLayer(1, display.OCRFrame{
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Layer 1 Text", Box: box}},
	})
	spatialFacet.SetLayer(2, display.OCRFrame{
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Layer 2 Text", Box: box}},
	})

	// Run multiple times to verify deterministic hit-testing order (should consistently hit Layer 1 first)
	for i := 0; i < 50; i++ {
		text, hit := spatialFacet.FindTextAt(geometry.Point{X: 50, Y: 50})
		if !hit {
			t.Fatalf("iteration %d: expected hit", i)
		}
		if text != "Layer 1 Text" {
			t.Fatalf("iteration %d: expected deterministic hit 'Layer 1 Text', got %q", i, text)
		}
	}

	// Verify AllTexts deterministic order: [Layer 1, Layer 2, Layer 3]
	for i := 0; i < 50; i++ {
		texts := spatialFacet.AllTexts()
		if len(texts) != 3 {
			t.Fatalf("iteration %d: expected 3 texts, got %d", i, len(texts))
		}
		if texts[0] != "Layer 1 Text" || texts[1] != "Layer 2 Text" || texts[2] != "Layer 3 Text" {
			t.Fatalf("iteration %d: expected [Layer 1 Text, Layer 2 Text, Layer 3 Text], got %v", i, texts)
		}
	}
}

func TestDesktopState_DisplayTopologyEvent_NegativeCoordinatesAndNonStandardResolutions(t *testing.T) {
	state := desktop.NewState()

	rectPort, _ := geometry.NewRectangle(1954, -708, 1080, 1920)
	dPort, _ := display.NewDisplay(1, rectPort, 1.0, false)

	rect4K, _ := geometry.NewRectangle(3050, -706, 3840, 2160)
	d4K, _ := display.NewDisplay(2, rect4K, 1.5, false)

	state.Apply(events.DisplayTopologyEvent{
		Timestamp: time.Now(),
		Displays:  []display.Display{dPort, d4K},
	})

	boxPort, _ := geometry.NewRectangle(100, 200, 300, 50)
	state.Apply(events.OCREvent{
		DisplayID:   1,
		Resolution:  geometry.Size{Width: 1080, Height: 1920},
		ScaleFactor: 1.0,
		WindowRect:  rectPort,
		Blocks:      []display.OCRTextBlock{{Text: "Portrait Header", Box: boxPort}},
	})

	box4K, _ := geometry.NewRectangle(200, 300, 500, 60)
	state.Apply(events.OCREvent{
		DisplayID:   2,
		Resolution:  geometry.Size{Width: 3840, Height: 2160},
		ScaleFactor: 1.5,
		WindowRect:  rect4K,
		Blocks:      []display.OCRTextBlock{{Text: "4K Invoice Details", Box: box4K}},
	})

	// Hit-test portrait monitor: global X = 1954 + 150 = 2104, Y = -708 + 220 = -488
	textPort, hitPort := state.FindTextAt(geometry.Point{X: 2104, Y: -488})
	if !hitPort || textPort != "Portrait Header" {
		t.Errorf("portrait monitor hit failed: text=%q, hit=%v", textPort, hitPort)
	}

	// Hit-test 4K monitor: global X = 3050 + 300 = 3350, Y = -706 + 320 = -386
	text4K, hit4K := state.FindTextAt(geometry.Point{X: 3350, Y: -386})
	if !hit4K || text4K != "4K Invoice Details" {
		t.Errorf("4K monitor hit failed: text=%q, hit=%v", text4K, hit4K)
	}

	// Miss on negative coordinates outside any block
	_, hitMiss := state.FindTextAt(geometry.Point{X: 2000, Y: -100})
	if hitMiss {
		t.Errorf("expected miss on empty negative coordinates")
	}
}

func TestDesktopState_MultiDisplay_DeduplicationLifecycle(t *testing.T) {
	d0, _ := display.NewDisplay(0, geometry.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080}, 1.0, true)
	d1, _ := display.NewDisplay(1, geometry.Rectangle{X: 1920, Y: 0, Width: 1920, Height: 1080}, 1.0, false)
	state := desktop.NewState(d0, d1)
	now := time.Now()

	// T1: Display 1 receives an OCR frame with "Delete Account"
	state.Apply(events.OCREvent{
		DisplayID:  1,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Filename:   "frame_d1_1.jpg",
		Blocks:     []display.OCRTextBlock{{Text: "Delete Account", Box: geometry.Rectangle{X: 100, Y: 100, Width: 200, Height: 50}}},
		Timestamp:  now,
	})
	if txt, hit := state.FindTextAt(geometry.Point{X: 2070, Y: 120}); !hit || txt != "Delete Account" {
		t.Fatalf("expected hit on Display 1 at T1, got %q, %v", txt, hit)
	}

	// T2: Display 0 receives a new OCR frame, Display 1 sends deduplication
	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Filename:   "frame_d0_2.jpg",
		Blocks:     []display.OCRTextBlock{{Text: "Display 0 New Text", Box: geometry.Rectangle{X: 50, Y: 50, Width: 100, Height: 40}}},
		Timestamp:  now.Add(time.Second),
	})
	state.Apply(events.OCREvent{
		DisplayID:        1,
		Resolution:       geometry.Size{Width: 1920, Height: 1080},
		Filename:         "frame_d1_2.jpg",
		DeduplicatedFrom: "frame_d1_1.jpg",
		Timestamp:        now.Add(time.Second),
	})

	if txt, hit := state.FindTextAt(geometry.Point{X: 2070, Y: 120}); !hit || txt != "Delete Account" {
		t.Fatalf("expected hit on Display 1 at T2 after deduplication, got %q, %v", txt, hit)
	}
	if txt0, hit0 := state.FindTextAt(geometry.Point{X: 80, Y: 60}); !hit0 || txt0 != "Display 0 New Text" {
		t.Fatalf("expected hit on Display 0 at T2, got %q, %v", txt0, hit0)
	}

	// T3: Screen cleared without deduplication
	state.Apply(events.OCREvent{
		DisplayID:  1,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Filename:   "frame_d1_3.jpg",
		Timestamp:  now.Add(2 * time.Second),
	})
	if _, hitCleared := state.FindTextAt(geometry.Point{X: 2070, Y: 120}); hitCleared {
		t.Fatalf("expected miss on Display 1 after screen cleared")
	}
}

func TestDesktopState_QuadMonitor_FractionalDPI_HitTesting(t *testing.T) {
	d0, _ := display.NewDisplay(0, geometry.Rectangle{X: 0, Y: 0, Width: 5120, Height: 1440}, 1.0, true)
	d1, _ := display.NewDisplay(1, geometry.Rectangle{X: -1920, Y: -1080, Width: 1920, Height: 1080}, 1.0, false)
	d2, _ := display.NewDisplay(2, geometry.Rectangle{X: 0, Y: -1440, Width: 2560, Height: 1440}, 1.25, false)
	d3, _ := display.NewDisplay(3, geometry.Rectangle{X: -3840, Y: 0, Width: 3840, Height: 2160}, 1.5, false)
	state := desktop.NewState(d0, d1, d2, d3)
	now := time.Now()

	state.Apply(events.OCREvent{
		DisplayID:  0,
		Resolution: geometry.Size{Width: 5120, Height: 1440},
		Blocks:     []display.OCRTextBlock{{Text: "Super UltraWide Right Wing", Box: geometry.Rectangle{X: 4000, Y: 500, Width: 600, Height: 100}}},
		Timestamp:  now,
	})
	state.Apply(events.OCREvent{
		DisplayID:  1,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks:     []display.OCRTextBlock{{Text: "Top-Left Negative Monitor", Box: geometry.Rectangle{X: 420, Y: 280, Width: 300, Height: 60}}},
		Timestamp:  now,
	})
	state.Apply(events.OCREvent{
		DisplayID:  2,
		Resolution: geometry.Size{Width: 2560, Height: 1440},
		Blocks:     []display.OCRTextBlock{{Text: "Top-Right QHD Monitor", Box: geometry.Rectangle{X: 1000, Y: 440, Width: 400, Height: 80}}},
		Timestamp:  now,
	})
	state.Apply(events.OCREvent{
		DisplayID:  3,
		Resolution: geometry.Size{Width: 3840, Height: 2160},
		Blocks:     []display.OCRTextBlock{{Text: "Bottom-Left 4K Monitor", Box: geometry.Rectangle{X: 1840, Y: 1000, Width: 500, Height: 120}}},
		Timestamp:  now,
	})

	assertHit := func(x, y int, want string) {
		t.Helper()
		if txt, hit := state.FindTextAt(geometry.Point{X: x, Y: y}); !hit || txt != want {
			t.Errorf("hit at (%d,%d) failed: got %q, %v, want %q", x, y, txt, hit, want)
		}
	}
	assertHit(4100, 520, "Super UltraWide Right Wing")
	assertHit(-1470, -780, "Top-Left Negative Monitor")
	assertHit(1050, -980, "Top-Right QHD Monitor")
	assertHit(-1940, 1050, "Bottom-Left 4K Monitor")
}
