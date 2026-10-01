package rules

import (
	"testing"
)

func TestTemplateCreation(t *testing.T) {
	_, err := NewPopupTemplate("", "body text")
	if err != ErrEmptyTemplateTitle {
		t.Fatalf("expected ErrEmptyTemplateTitle, got: %v", err)
	}

	tmpl, err := NewPopupTemplate("Forwarded email", "You opened a forwarded email: {click}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vars := map[string]string{
		"click": "FW: Floor plan",
	}

	title, body := tmpl.Render(vars)
	if title != "Forwarded email" {
		t.Fatalf("unexpected title: %s", title)
	}
	if body != "You opened a forwarded email: FW: Floor plan" {
		t.Fatalf("unexpected body: %s", body)
	}
}
