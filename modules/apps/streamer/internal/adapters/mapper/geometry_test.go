package mapper_test

import (
	"testing"

	"assessment/modules/apps/streamer/internal/adapters/mapper"
	"assessment/modules/libs/domain/geometry"
)

func TestToProtoPoint(t *testing.T) {
	p := geometry.Point{X: 123, Y: 456}
	pb := mapper.ToProtoPoint(p)
	if pb.X != 123 || pb.Y != 456 {
		t.Errorf("got point (%d, %d), want (123, 456)", pb.X, pb.Y)
	}
}

func TestToProtoSize(t *testing.T) {
	s := geometry.Size{Width: 1920, Height: 1080}
	pb := mapper.ToProtoSize(s)
	if pb.Width != 1920 || pb.Height != 1080 {
		t.Errorf("got size (%d, %d), want (1920, 1080)", pb.Width, pb.Height)
	}
}

func TestToProtoRectangle(t *testing.T) {
	r := geometry.Rectangle{X: 10, Y: 20, Width: 300, Height: 400}
	pb := mapper.ToProtoRectangle(r)
	if pb.X != 10 || pb.Y != 20 || pb.Width != 300 || pb.Height != 400 {
		t.Errorf("got rect %v, want (10, 20, 300, 400)", pb)
	}
}
