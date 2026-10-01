package audit

import (
	"testing"
	"time"
)

func TestPopupValidation(t *testing.T) {
	_, err := NewPopup("", "rule-1", "2026-03-10T23:15:21Z", "Title", "Body")
	if err != ErrEmptyPopupEmployee {
		t.Fatalf("expected ErrEmptyPopupEmployee, got: %v", err)
	}

	_, err = NewPopup("emp-1", "", "2026-03-10T23:15:21Z", "Title", "Body")
	if err != ErrEmptyPopupRule {
		t.Fatalf("expected ErrEmptyPopupRule, got: %v", err)
	}

	_, err = NewPopup("emp-1", "rule-1", "", "Title", "Body")
	if err != ErrEmptyPopupTS {
		t.Fatalf("expected ErrEmptyPopupTS, got: %v", err)
	}

	_, err = NewPopup("emp-1", "rule-1", "2026-03-10T23:15:21Z", "", "Body")
	if err != ErrEmptyPopupTitle {
		t.Fatalf("expected ErrEmptyPopupTitle, got: %v", err)
	}

	popup, err := NewPopupWithTime("emp-1", "forwarded-email-opened", time.Now(), "Forwarded email", "You opened an email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if popup.Employee != "emp-1" || popup.Rule != "forwarded-email-opened" {
		t.Fatalf("unexpected popup values: %+v", popup)
	}
}
