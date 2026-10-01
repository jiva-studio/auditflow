package mapper

import (
	"testing"
	"time"

	"assessment/modules/libs/domain/audit"
)

func TestToProtoPopup(t *testing.T) {
	t.Run("valid RFC3339Nano popup", func(t *testing.T) {
		ts := time.Now().UTC().Format(time.RFC3339Nano)
		domainPopup, err := audit.NewPopup("emp-1", "rule-1", ts, "Test Title", "Test Body")
		if err != nil {
			t.Fatalf("unexpected error creating popup: %v", err)
		}

		pb := ToProtoPopup(domainPopup)
		if pb.GetEmployee() != "emp-1" || pb.GetRule() != "rule-1" || pb.GetTitle() != "Test Title" || pb.GetBody() != "Test Body" {
			t.Errorf("unexpected proto popup: %+v", pb)
		}
		if pb.GetTs() == nil {
			t.Error("expected non-nil proto timestamp")
		}
	})

	t.Run("valid RFC3339 popup", func(t *testing.T) {
		ts := time.Now().UTC().Format(time.RFC3339)
		domainPopup, err := audit.NewPopup("emp-2", "rule-2", ts, "T2", "B2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		pb := ToProtoPopup(domainPopup)
		if pb.GetEmployee() != "emp-2" {
			t.Errorf("unexpected employee: %s", pb.GetEmployee())
		}
	})

	t.Run("invalid timestamp string fallback", func(t *testing.T) {
		p := audit.Popup{
			Employee: "emp-3",
			Rule:     "rule-3",
			TS:       "not-a-valid-timestamp",
			Title:    "T3",
			Body:     "B3",
		}
		pb := ToProtoPopup(p)
		if pb.GetTs() == nil {
			t.Error("expected fallback timestamp to be non-nil")
		}
	})
}
