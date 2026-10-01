// Package display provides models for displays, OCR frames, and hit-testing.
package display

import (
	"errors"
	"time"

	"assessment/libs/domain/geometry"
)

// Domain errors for display validation.
var (
	ErrInvalidDisplayScale  = errors.New("display scale must be positive")
	ErrInvalidDisplayBounds = errors.New("display bounds width and height must be positive")
)

// OCRTextBlock represents a single recognized text block with its bounding box in OCR image space.
type OCRTextBlock struct {
	Text       string
	Box        geometry.Rectangle
	Confidence float64
}

// Contains checks if a local OCR coordinate point falls within this text block.
func (b OCRTextBlock) Contains(localPoint geometry.Point) bool {
	return b.Box.Contains(localPoint)
}

// OCRFrame represents an OCR snapshot of a specific display.
type OCRFrame struct {
	DisplayID   int
	Resolution  geometry.Size
	ScaleFactor float64
	Blocks      []OCRTextBlock
	Timestamp   time.Time
	Filename    string
}

// SpatialHit represents the result of a spatial query against a visual layer.
type SpatialHit struct {
	Text       string
	Confidence float64
	Source     string
}

// SpatialLayer represents a queryable layer on a display (e.g. OCR, Accessibility tree, DOM).
type SpatialLayer interface {
	HitTest(localPoint geometry.Point, displayBounds geometry.Rectangle) (SpatialHit, bool)
	AllTexts() []string
}

// FindBlockAt finds the first text block containing the given local display point.
func (f OCRFrame) FindBlockAt(localPoint geometry.Point, displayBounds geometry.Rectangle) (OCRTextBlock, bool) {
	if f.Resolution.Width <= 0 || f.Resolution.Height <= 0 {
		return OCRTextBlock{}, false
	}

	scaleX := float64(displayBounds.Width) / float64(f.Resolution.Width)
	scaleY := float64(displayBounds.Height) / float64(f.Resolution.Height)

	for _, block := range f.Blocks {
		scaledBox := geometry.Rectangle{
			X:      int(float64(block.Box.X) * scaleX),
			Y:      int(float64(block.Box.Y) * scaleY),
			Width:  int(float64(block.Box.Width) * scaleX),
			Height: int(float64(block.Box.Height) * scaleY),
		}
		if scaledBox.Contains(localPoint) {
			return block, true
		}
	}
	return OCRTextBlock{}, false
}

// HitTest implements SpatialLayer for OCRFrame.
func (f OCRFrame) HitTest(localPoint geometry.Point, displayBounds geometry.Rectangle) (SpatialHit, bool) {
	block, hit := f.FindBlockAt(localPoint, displayBounds)
	if !hit {
		return SpatialHit{}, false
	}
	return SpatialHit{
		Text:       block.Text,
		Confidence: block.Confidence,
		Source:     "ocr",
	}, true
}

// AllTexts returns all non-empty recognized text strings from this OCR frame.
func (f OCRFrame) AllTexts() []string {
	var texts []string
	for _, block := range f.Blocks {
		if block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return texts
}

// Display represents a physical/virtual display with its bounding box and DPI scale.
type Display struct {
	ID      int
	Bounds  geometry.Rectangle
	Scale   float64
	Primary bool
}

// NewDisplay creates and validates a Display value object.
func NewDisplay(id int, bounds geometry.Rectangle, scale float64, primary bool) (Display, error) {
	if scale <= 0 {
		return Display{}, ErrInvalidDisplayScale
	}
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return Display{}, ErrInvalidDisplayBounds
	}
	return Display{
		ID:      id,
		Bounds:  bounds,
		Scale:   scale,
		Primary: primary,
	}, nil
}

// Contains checks if global virtual desktop coordinates fall on this display.
func (d Display) Contains(globalPoint geometry.Point) bool {
	return d.Bounds.Contains(globalPoint)
}

// MapToLocal converts global virtual desktop coordinates to local display-relative coordinates.
func (d Display) MapToLocal(globalPoint geometry.Point) (geometry.Point, bool) {
	if !d.Contains(globalPoint) {
		return geometry.Point{}, false
	}
	return geometry.Point{
		X: globalPoint.X - d.Bounds.X,
		Y: globalPoint.Y - d.Bounds.Y,
	}, true
}

// HitTest finds the OCR text block at the global point on this display.
func (d Display) HitTest(globalPoint geometry.Point, frame OCRFrame) (OCRTextBlock, bool) {
	localPoint, ok := d.MapToLocal(globalPoint)
	if !ok {
		return OCRTextBlock{}, false
	}
	return frame.FindBlockAt(localPoint, d.Bounds)
}
