// Package mapper converts domain models to protocol buffer representations.
package mapper

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/modules/libs/domain/events"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

// ToProtoTickBatch converts a domain TickBatch aggregate into a Protobuf TickBatch message.
func ToProtoTickBatch(batch events.TickBatch) *v1.TickBatch {
	pb := &v1.TickBatch{
		TickIndex: int32(batch.TickIndex),
		StartTime: timestamppb.New(batch.StartTime),
		EndTime:   timestamppb.New(batch.EndTime),
		Events:    make([]*v1.Event, 0, batch.Len()),
	}

	for _, ev := range batch.Events() {
		if pbEv := ToProtoEvent(ev); pbEv != nil {
			pb.Events = append(pb.Events, pbEv)
		}
	}

	return pb
}
