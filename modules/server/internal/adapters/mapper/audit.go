// Package mapper provides conversion between Protobuf contracts, JSON DTOs, and domain audit entities.
package mapper

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/libs/domain/audit"
	v1 "assessment/libs/protocol/gen/go/v1"
)

// Sentinel errors for audit mapper.
var (
	ErrNilProtoPopup = errors.New("proto popup cannot be nil")
)

// PopupDTO represents the JSON transfer representation of an audit popup.
type PopupDTO struct {
	Employee string `json:"employee"`
	Rule     string `json:"rule"`
	TS       string `json:"ts"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

// ToDomainPopup converts a Protobuf v1.Popup to a domain audit.Popup.
func ToDomainPopup(pb *v1.Popup) (audit.Popup, error) {
	if pb == nil {
		return audit.Popup{}, ErrNilProtoPopup
	}

	var ts time.Time
	if pb.GetTs() != nil {
		ts = pb.GetTs().AsTime()
	} else {
		ts = time.Now().UTC()
	}

	return audit.NewPopupWithTime(
		pb.GetEmployee(),
		pb.GetRule(),
		ts,
		pb.GetTitle(),
		pb.GetBody(),
	)
}

// ToProtoPopup converts a domain audit.Popup to a Protobuf v1.Popup.
func ToProtoPopup(p audit.Popup) (*v1.Popup, error) {
	t, err := time.Parse(time.RFC3339Nano, p.TS)
	if err != nil {
		t, err = time.Parse(time.RFC3339, p.TS)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", p.TS, err)
		}
	}

	return &v1.Popup{
		Employee: p.Employee,
		Rule:     p.Rule,
		Ts:       timestamppb.New(t),
		Title:    p.Title,
		Body:     p.Body,
	}, nil
}

// ToProtoAuditResponse converts a slice of domain popups to a Protobuf v1.AuditResponse.
func ToProtoAuditResponse(popups []audit.Popup) (*v1.AuditResponse, error) {
	pbPopups := make([]*v1.Popup, 0, len(popups))
	for _, p := range popups {
		pb, err := ToProtoPopup(p)
		if err != nil {
			return nil, err
		}
		pbPopups = append(pbPopups, pb)
	}
	return &v1.AuditResponse{Popups: pbPopups}, nil
}

// ToDTO converts a domain audit.Popup to PopupDTO.
func ToDTO(p audit.Popup) PopupDTO {
	return PopupDTO{
		Employee: p.Employee,
		Rule:     p.Rule,
		TS:       p.TS,
		Title:    p.Title,
		Body:     p.Body,
	}
}

// ToDTOList converts a slice of domain popups to a slice of PopupDTO.
func ToDTOList(popups []audit.Popup) []PopupDTO {
	dtos := make([]PopupDTO, 0, len(popups))
	for _, p := range popups {
		dtos = append(dtos, ToDTO(p))
	}
	return dtos
}
