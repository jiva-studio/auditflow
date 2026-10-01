package geometry

import (
	"testing"
)

func TestPoint(t *testing.T) {
	p := Point{X: 10, Y: 20}
	p2 := p.Offset(5, -10)
	if p2.X != 15 || p2.Y != 10 {
		t.Fatalf("unexpected offset point: %+v", p2)
	}

	rect, err := NewRectangle(0, 0, 100, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.In(rect) {
		t.Fatalf("point %+v should be in rect %+v", p, rect)
	}
	outsidePoint := Point{X: 100, Y: 100}
	if outsidePoint.In(rect) {
		t.Fatalf("point (100, 100) should not be inside half-open bounds [0, 100)")
	}
}

func TestRectangleValidation(t *testing.T) {
	_, err := NewRectangle(0, 0, -1, 10)
	if err != ErrInvalidDimensions {
		t.Fatalf("expected ErrInvalidDimensions, got: %v", err)
	}
}

func TestRectangleContains(t *testing.T) {
	r, _ := NewRectangle(10, 20, 100, 50)
	if !r.Contains(Point{X: 10, Y: 20}) || !r.Contains(Point{X: 50, Y: 40}) {
		t.Fatal("points inside should return true")
	}
	if r.Contains(Point{X: 110, Y: 70}) || r.Contains(Point{X: 9, Y: 20}) {
		t.Fatal("points outside should return false")
	}
}

func TestRectangleScale(t *testing.T) {
	r, _ := NewRectangle(10, 20, 100, 50)
	scaled := r.Scale(1.5, 2.0)
	if scaled.X != 15 || scaled.Y != 40 || scaled.Width != 150 || scaled.Height != 100 {
		t.Fatalf("unexpected scaled rect: %+v", scaled)
	}
}

func TestRectangleIntersects(t *testing.T) {
	r, _ := NewRectangle(10, 20, 100, 50)
	r2, _ := NewRectangle(50, 30, 100, 100)
	if !r.Intersects(r2) {
		t.Fatal("rectangles should intersect")
	}
	r3, _ := NewRectangle(200, 200, 50, 50)
	if r.Intersects(r3) {
		t.Fatal("rectangles should not intersect")
	}
}
