// Package mapper converts domain models to protocol buffer representations.
package mapper

import (
	"assessment/modules/libs/domain/geometry"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

// ToProtoPoint converts a domain Point to a Protobuf Point.
func ToProtoPoint(p geometry.Point) *v1.Point {
	return &v1.Point{
		X: int32(p.X),
		Y: int32(p.Y),
	}
}

// ToProtoSize converts a domain Size to a Protobuf Size.
func ToProtoSize(s geometry.Size) *v1.Size {
	return &v1.Size{
		Width:  int32(s.Width),
		Height: int32(s.Height),
	}
}

// ToProtoRectangle converts a domain Rectangle to a Protobuf Rectangle.
func ToProtoRectangle(r geometry.Rectangle) *v1.Rectangle {
	return &v1.Rectangle{
		X:      int32(r.X),
		Y:      int32(r.Y),
		Width:  int32(r.Width),
		Height: int32(r.Height),
	}
}
