// Package geometry defines 2D integer points, sizes, and bounding boxes.
package geometry

import (
	"errors"
)

// ErrInvalidDimensions indicates width or height is negative.
var ErrInvalidDimensions = errors.New("dimensions must be non-negative")

// Point represents a 2D integer coordinate on a screen.
type Point struct {
	X int
	Y int
}

// Offset returns a new Point translated by (dx, dy).
func (p Point) Offset(dx, dy int) Point {
	return Point{X: p.X + dx, Y: p.Y + dy}
}

// In returns true if the point lies inside the given Rectangle.
func (p Point) In(r Rectangle) bool {
	return r.Contains(p)
}

// Size represents 2D dimensions.
type Size struct {
	Width  int
	Height int
}

// Validate checks that dimensions are valid.
func (s Size) Validate() error {
	if s.Width < 0 || s.Height < 0 {
		return ErrInvalidDimensions
	}
	return nil
}

// Rectangle represents a 2D rectangle defined by top-left origin and size.
type Rectangle struct {
	X      int
	Y      int
	Width  int
	Height int
}

// NewRectangle creates and validates a new Rectangle.
func NewRectangle(x, y, width, height int) (Rectangle, error) {
	if width < 0 || height < 0 {
		return Rectangle{}, ErrInvalidDimensions
	}
	return Rectangle{X: x, Y: y, Width: width, Height: height}, nil
}

// Contains checks if a point p is within the bounds [X, X+Width) and [Y, Y+Height).
func (r Rectangle) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.Width &&
		p.Y >= r.Y && p.Y < r.Y+r.Height
}

// Scale scales the rectangle by scale factors along X and Y axes.
func (r Rectangle) Scale(scaleX, scaleY float64) Rectangle {
	return Rectangle{
		X:      int(float64(r.X) * scaleX),
		Y:      int(float64(r.Y) * scaleY),
		Width:  int(float64(r.Width) * scaleX),
		Height: int(float64(r.Height) * scaleY),
	}
}

// Intersects checks if this rectangle overlaps with another rectangle.
func (r Rectangle) Intersects(other Rectangle) bool {
	return r.X < other.X+other.Width &&
		r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height &&
		r.Y+r.Height > other.Y
}
