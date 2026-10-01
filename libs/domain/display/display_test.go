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
