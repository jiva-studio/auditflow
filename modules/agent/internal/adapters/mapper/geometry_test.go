package mapper

import (
	"testing"

	"assessment/libs/domain/display"
	"assessment/libs/domain/geometry"
	v1 "assessment/libs/protocol/gen/go/v1"
)

func TestToDomainPoint(t *testing.T) {
	t.Run("nil point", func(t *testing.T) {
		pt := ToDomainPoint(nil)
		if pt != (geometry.Point{}) {
			t.Errorf("expected empty point, got %+v", pt)
		}
	})

	t.Run("valid point", func(t *testing.T) {
		pb := &v1.Point{X: 100, Y: 200}
		pt := ToDomainPoint(pb)
		if pt.X != 100 || pt.Y != 200 {
			t.Errorf("expected (100, 200), got %+v", pt)
		}
	})
}

func TestToDomainSize(t *testing.T) {
	t.Run("nil size", func(t *testing.T) {
		sz := ToDomainSize(nil)
		if sz != (geometry.Size{}) {
			t.Errorf("expected empty size, got %+v", sz)
		}
	})

	t.Run("valid size", func(t *testing.T) {
		pb := &v1.Size{Width: 1920, Height: 1080}
		sz := ToDomainSize(pb)
		if sz.Width != 1920 || sz.Height != 1080 {
			t.Errorf("expected 1920x1080, got %+v", sz)
		}
	})
}

func TestToDomainRectangle(t *testing.T) {
	t.Run("nil rectangle", func(t *testing.T) {
		rect := ToDomainRectangle(nil)
		if rect != (geometry.Rectangle{}) {
			t.Errorf("expected empty rectangle, got %+v", rect)
		}
	})

	t.Run("valid rectangle", func(t *testing.T) {
		pb := &v1.Rectangle{X: 10, Y: 20, Width: 300, Height: 400}
		rect := ToDomainRectangle(pb)
		expected := geometry.Rectangle{X: 10, Y: 20, Width: 300, Height: 400}
		if rect != expected {
			t.Errorf("expected %+v, got %+v", expected, rect)
		}
	})
}

func TestToDomainOCRTextBlock(t *testing.T) {
	t.Run("nil text block", func(t *testing.T) {
		tb := ToDomainOCRTextBlock(nil)
		if tb != (display.OCRTextBlock{}) {
			t.Errorf("expected empty text block, got %+v", tb)
		}
	})

	t.Run("valid text block", func(t *testing.T) {
		pb := &v1.OCRTextBlock{
			Text:       "Inbox",
			Box:        &v1.Rectangle{X: 1, Y: 2, Width: 30, Height: 40},
			Confidence: 0.95,
		}
		tb := ToDomainOCRTextBlock(pb)
		if tb.Text != "Inbox" || tb.Confidence != 0.95 || tb.Box.X != 1 {
			t.Errorf("unexpected text block: %+v", tb)
		}
	})
}
