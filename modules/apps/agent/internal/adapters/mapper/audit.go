package mapper

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/modules/libs/domain/audit"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

// ToProtoPopup converts a domain audit.Popup entity into a Protobuf Popup message.
func ToProtoPopup(p audit.Popup) *v1.Popup {
	var pbTS *timestamppb.Timestamp
	if parsedTime, err := time.Parse(time.RFC3339Nano, p.TS); err == nil {
		pbTS = timestamppb.New(parsedTime)
	} else if parsedTime, err := time.Parse(time.RFC3339, p.TS); err == nil {
		pbTS = timestamppb.New(parsedTime)
	} else {
		pbTS = timestamppb.Now()
	}

	return &v1.Popup{
		Employee: p.Employee,
		Rule:     p.Rule,
		Ts:       pbTS,
		Title:    p.Title,
		Body:     p.Body,
	}
}
