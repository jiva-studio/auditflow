// Package mapper converts protocol buffer representations to domain models and vice versa.
package mapper

import (
	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/geometry"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

// ToDomainPoint converts a Protobuf Point to a domain Point.
func ToDomainPoint(p *v1.Point) geometry.Point {
	if p == nil {
		return geometry.Point{}
	}
	return geometry.Point{
		X: int(p.GetX()),
		Y: int(p.GetY()),
	}
}

// ToDomainSize converts a Protobuf Size to a domain Size.
func ToDomainSize(s *v1.Size) geometry.Size {
	if s == nil {
		return geometry.Size{}
	}
	return geometry.Size{
		Width:  int(s.GetWidth()),
		Height: int(s.GetHeight()),
	}
}

// ToDomainRectangle converts a Protobuf Rectangle to a domain Rectangle.
func ToDomainRectangle(r *v1.Rectangle) geometry.Rectangle {
	if r == nil {
		return geometry.Rectangle{}
	}
	return geometry.Rectangle{
		X:      int(r.GetX()),
		Y:      int(r.GetY()),
		Width:  int(r.GetWidth()),
		Height: int(r.GetHeight()),
	}
}

// ToDomainOCRTextBlock converts a Protobuf OCRTextBlock to a domain OCRTextBlock.
func ToDomainOCRTextBlock(b *v1.OCRTextBlock) display.OCRTextBlock {
	if b == nil {
		return display.OCRTextBlock{}
	}
	return display.OCRTextBlock{
		Text:       b.GetText(),
		Box:        ToDomainRectangle(b.GetBox()),
		Confidence: b.GetConfidence(),
	}
}
