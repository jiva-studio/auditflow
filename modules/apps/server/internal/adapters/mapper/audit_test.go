package mapper_test

import (
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/modules/apps/server/internal/adapters/mapper"
	"assessment/modules/libs/domain/audit"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

func TestToDomainPopup_Success(t *testing.T) {
	fixedTime := time.Date(2026, 3, 10, 15, 30, 0, 0, time.UTC)
	pb := &v1.Popup{
		Employee: "emp-100",
		Rule:     "forwarded-email-opened",
		Ts:       timestamppb.New(fixedTime),
		Title:    "Forwarded email",
		Body:     "Body text",
	}

	p, err := mapper.ToDomainPopup(pb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Employee != "emp-100" || p.Rule != "forwarded-email-opened" || p.Title != "Forwarded email" {
		t.Fatalf("mapped fields mismatch: %+v", p)
	}
}

func TestToDomainPopup_NilProto(t *testing.T) {
	_, err := mapper.ToDomainPopup(nil)
	if !errors.Is(err, mapper.ErrNilProtoPopup) {
		t.Fatalf("expected ErrNilProtoPopup, got %v", err)
	}
}

func TestToDomainPopup_NilTimestamp(t *testing.T) {
	pb := &v1.Popup{
		Employee: "emp-100",
		Rule:     "forwarded-email-opened",
		Ts:       nil,
		Title:    "Forwarded email",
		Body:     "Body text",
	}

	p, err := mapper.ToDomainPopup(pb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.TS == "" {
		t.Fatal("expected non-empty timestamp")
	}
}

func TestToDomainPopup_InvalidDomainPopup(t *testing.T) {
	pb := &v1.Popup{
		Employee: "",
		Rule:     "rule-1",
		Title:    "Title",
	}
	_, err := mapper.ToDomainPopup(pb)
	if err == nil {
		t.Fatal("expected error on empty employee")
	}
}

func TestToProtoPopup_Success(t *testing.T) {
	p, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title", "Body")
	pb, err := mapper.ToProtoPopup(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pb.GetEmployee() != "emp-1" || pb.GetRule() != "rule-1" {
		t.Fatalf("proto fields mismatch: %+v", pb)
	}
}

func TestToProtoPopup_InvalidTimestamp(t *testing.T) {
	p := audit.Popup{
		Employee: "emp-1",
		Rule:     "rule-1",
		TS:       "invalid-time",
		Title:    "Title",
	}
	_, err := mapper.ToProtoPopup(p)
	if err == nil {
		t.Fatal("expected error on invalid timestamp")
	}
}

func TestToProtoAuditResponse_Success(t *testing.T) {
	p1, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title 1", "Body 1")
	p2, _ := audit.NewPopup("emp-2", "rule-2", "2026-03-10T13:00:00Z", "Title 2", "Body 2")

	resp, err := mapper.ToProtoAuditResponse([]audit.Popup{p1, p2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetPopups()) != 2 {
		t.Fatalf("expected 2 popups in proto response, got %d", len(resp.GetPopups()))
	}
}

func TestToProtoAuditResponse_Error(t *testing.T) {
	badPopup := audit.Popup{
		Employee: "emp-1",
		Rule:     "rule-1",
		TS:       "bad-time",
		Title:    "Title",
	}
	_, err := mapper.ToProtoAuditResponse([]audit.Popup{badPopup})
	if err == nil {
		t.Fatal("expected error with malformed popup timestamp")
	}
}

func TestToDTOAndToDTOList(t *testing.T) {
	p1, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title 1", "Body 1")
	p2, _ := audit.NewPopup("emp-2", "rule-2", "2026-03-10T13:00:00Z", "Title 2", "Body 2")

	dto1 := mapper.ToDTO(p1)
	if dto1.Employee != "emp-1" || dto1.Rule != "rule-1" || dto1.Title != "Title 1" {
		t.Fatalf("mismatched DTO contents: %+v", dto1)
	}

	list := mapper.ToDTOList([]audit.Popup{p1, p2})
	if len(list) != 2 || list[0].Employee != "emp-1" || list[1].Employee != "emp-2" {
		t.Fatalf("mismatched DTO list contents: %+v", list)
	}

	emptyList := mapper.ToDTOList(nil)
	if emptyList == nil || len(emptyList) != 0 {
		t.Fatalf("expected empty non-nil slice, got: %+v", emptyList)
	}
}
