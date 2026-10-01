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
}
