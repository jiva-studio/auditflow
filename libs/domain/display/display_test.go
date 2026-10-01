package display

import (
	"testing"
	"time"

	"assessment/libs/domain/geometry"
)

func TestDisplayCreation(t *testing.T) {
	rect, _ := geometry.NewRectangle(0, 0, 1920, 1080)
	_, err := NewDisplay(1, rect, -1.0, true)
	if err != ErrInvalidDisplayScale {
		t.Fatalf("expected ErrInvalidDisplayScale, got: %v", err)
	}

	invalidRect, _ := geometry.NewRectangle(0, 0, 0, 1080)
	_, err = NewDisplay(1, invalidRect, 1.5, true)
	if err != ErrInvalidDisplayBounds {
		t.Fatalf("expected ErrInvalidDisplayBounds, got: %v", err)
	}

	d, err := NewDisplay(2, geometry.Rectangle{X: 3050, Y: -706, Width: 3840, Height: 2160}, 1.5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Contains
	if !d.Contains(geometry.Point{X: 3050, Y: -706}) {
		t.Fatal("top-left should be inside display")
	}
	if !d.Contains(geometry.Point{X: 5000, Y: 0}) {
		t.Fatal("point inside should return true")
	}
	if d.Contains(geometry.Point{X: 1000, Y: 0}) {
		t.Fatal("point outside should return false")
	}

	// MapToLocal
	local, ok := d.MapToLocal(geometry.Point{X: 3832, Y: 400})
	if !ok {
		t.Fatal("expected mapping to succeed")
	}
	if local.X != 3832-3050 || local.Y != 400-(-706) {
		t.Fatalf("unexpected local point: %+v", local)
	}
}

func TestDisplayHitTest(t *testing.T) {
	d, err := NewDisplay(2, geometry.Rectangle{X: 3050, Y: -706, Width: 3840, Height: 2160}, 1.5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Display resolution is 1920x1080, display bounds width is 3840, height 2160 (scale 2.0 per axis)
	frame := OCRFrame{
		DisplayID:   2,
		Resolution:  geometry.Size{Width: 1920, Height: 1080},
		ScaleFactor: 1.5,
		Timestamp:   time.Now(),
		Blocks: []OCRTextBlock{
			{
				Text:       "FW: Updated floor plan",
				Box:        geometry.Rectangle{X: 100, Y: 200, Width: 300, Height: 50},
				Confidence: 0.99,
			},
		},
	}

	// Scaled box in display local coords: X: 200..800, Y: 400..500
	// Global coords = 3050 + 250 = 3300, Y = -706 + 450 = -256
	hitGlobal := geometry.Point{X: 3300, Y: -256}
	block, ok := d.HitTest(hitGlobal, frame)
	if !ok {
		t.Fatal("expected hit test to find block")
	}
	if block.Text != "FW: Updated floor plan" {
		t.Fatalf("unexpected block text: %s", block.Text)
	}

	// Miss hit
	missGlobal := geometry.Point{X: 3050 + 50, Y: -706 + 50}
	_, ok = d.HitTest(missGlobal, frame)
	if ok {
		t.Fatal("expected hit test to miss")
	}

	// HitTest outside display
	outsideDisplay := geometry.Point{X: 0, Y: 0}
	_, ok = d.HitTest(outsideDisplay, frame)
	if ok {
		t.Fatal("expected hit test outside display to fail")
	}

	// MapToLocal outside display
	_, ok = d.MapToLocal(outsideDisplay)
	if ok {
		t.Fatal("expected MapToLocal outside display to fail")
	}
}

func TestOCRTextBlock_Contains(t *testing.T) {
	d, _ := NewDisplay(1, geometry.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080}, 1.0, true)

	tb := OCRTextBlock{
		Text: "test",
		Box:  geometry.Rectangle{X: 10, Y: 10, Width: 50, Height: 20},
	}
	if !tb.Contains(geometry.Point{X: 20, Y: 15}) {
		t.Fatal("expected text block to contain point")
	}
	if tb.Contains(geometry.Point{X: 100, Y: 100}) {
		t.Fatal("expected text block not to contain point")
	}

	invalidFrame := OCRFrame{
		Resolution: geometry.Size{Width: 0, Height: 0},
	}
	_, ok := invalidFrame.FindBlockAt(geometry.Point{X: 10, Y: 10}, d.Bounds)
	if ok {
		t.Fatal("expected FindBlockAt to fail with zero resolution")
	}
}

func TestOCRFrame_SpatialLayer(t *testing.T) {
	d, _ := NewDisplay(1, geometry.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080}, 1.0, true)

	validFrame := OCRFrame{
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []OCRTextBlock{
			{Text: "Layer Text 1", Box: geometry.Rectangle{X: 10, Y: 10, Width: 100, Height: 50}, Confidence: 0.98},
			{Text: "", Box: geometry.Rectangle{X: 0, Y: 0, Width: 5, Height: 5}},
			{Text: "Layer Text 2", Box: geometry.Rectangle{X: 200, Y: 200, Width: 100, Height: 50}, Confidence: 0.95},
		},
	}

	var layer SpatialLayer = validFrame
	hit, found := layer.HitTest(geometry.Point{X: 20, Y: 20}, d.Bounds)
	if !found || hit.Text != "Layer Text 1" || hit.Source != "ocr" || hit.Confidence != 0.98 {
		t.Fatalf("unexpected hit: %+v, %v", hit, found)
	}

	_, found = layer.HitTest(geometry.Point{X: 500, Y: 500}, d.Bounds)
	if found {
		t.Fatal("expected hit test to miss")
	}

	texts := layer.AllTexts()
	if len(texts) != 2 || texts[0] != "Layer Text 1" || texts[1] != "Layer Text 2" {
		t.Fatalf("unexpected all texts: %v", texts)
	}
}

func TestDisplay_FractionalDPI_Scaling(t *testing.T) {
	// Display logical bounds: 2560x1440 at (1920, 0), Scale: 1.25
	// OCR physical screenshot resolution: 3200x1800 (1.25x of 2560x1440)
	dBounds := geometry.Rectangle{X: 1920, Y: 0, Width: 2560, Height: 1440}
	d, err := NewDisplay(1, dBounds, 1.25, false)
	if err != nil {
		t.Fatalf("failed to create display: %v", err)
	}

	frame := OCRFrame{
		DisplayID:   1,
		Resolution:  geometry.Size{Width: 3200, Height: 1800},
		ScaleFactor: 1.25,
		Timestamp:   time.Now(),
		Blocks: []OCRTextBlock{
			{
				Text:       "Fractional Scale Button",
				Box:        geometry.Rectangle{X: 1000, Y: 500, Width: 500, Height: 200}, // in physical image pixels
				Confidence: 0.99,
			},
		},
	}

	// Scaled box in logical display coordinates:
	// scale = 2560 / 3200 = 0.8
	// X: 1000 * 0.8 = 800, Y: 500 * 0.8 = 400, Width: 500 * 0.8 = 400, Height: 200 * 0.8 = 160
	// Logical range: X in [800, 1200], Y in [400, 560]
	// Global range: X in [1920+800, 1920+1200] = [2720, 3120], Y in [0+400, 0+560] = [400, 560]

	// 1. Point inside button at (2900, 480) -> should HIT
	hitGlobal := geometry.Point{X: 2900, Y: 480}
	block, ok := d.HitTest(hitGlobal, frame)
	if !ok || block.Text != "Fractional Scale Button" {
		t.Fatalf("expected hit on fractional scale button, got block=%+v, ok=%v", block, ok)
	}

	// 2. Point outside button at (2600, 480) -> should MISS
	missGlobal := geometry.Point{X: 2600, Y: 480}
	_, ok = d.HitTest(missGlobal, frame)
	if ok {
		t.Fatalf("expected miss outside button on fractional scale display")
	}
}

func TestDisplay_ExactBoundariesAndEdges(t *testing.T) {
	// 2 Adjacent displays:
	// Left: [-1920, 0, 1920, 1080]
	// Right: [0, 0, 1920, 1080]
	dLeft, _ := NewDisplay(1, geometry.Rectangle{X: -1920, Y: 0, Width: 1920, Height: 1080}, 1.0, false)
	dRight, _ := NewDisplay(0, geometry.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080}, 1.0, true)

	// Left monitor top-left corner
	if !dLeft.Contains(geometry.Point{X: -1920, Y: 0}) {
		t.Errorf("expected (-1920, 0) to be in left display")
	}
	// Left monitor bottom-right pixel: (-1, 1079)
	if !dLeft.Contains(geometry.Point{X: -1, Y: 1079}) {
		t.Errorf("expected (-1, 1079) to be in left display")
	}
	// (0, 0) belongs to Right monitor, NOT Left monitor
	if dLeft.Contains(geometry.Point{X: 0, Y: 0}) {
		t.Errorf("expected (0, 0) NOT to be in left display")
	}
	if !dRight.Contains(geometry.Point{X: 0, Y: 0}) {
		t.Errorf("expected (0, 0) to be in right display")
	}
	// Right monitor bottom-right pixel: (1919, 1079)
	if !dRight.Contains(geometry.Point{X: 1919, Y: 1079}) {
		t.Errorf("expected (1919, 1079) to be in right display")
	}
	// (1920, 1080) is outside both monitors
	if dRight.Contains(geometry.Point{X: 1920, Y: 1080}) {
		t.Errorf("expected (1920, 1080) to be outside right display")
	}
}
