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

func TestTemplate_PlaceholderSubstitution(t *testing.T) {
	tmpl, err := NewPopupTemplate(
		"Action on {window_title} ({process})",
		"Clicked {click} with clip {clipboard} and screen {ocr}. Repeats: {click}, untouched: {missing}",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vars := map[string]string{
		"click":        "Submit Button",
		"process":      "chrome.exe",
		"window_title": "Admin Dashboard",
		"clipboard":    "TOKEN-12345",
		"ocr":          "Confirmation Message",
	}

	title, body := tmpl.Render(vars)
	expectedTitle := "Action on Admin Dashboard (chrome.exe)"
	expectedBody := "Clicked Submit Button with clip TOKEN-12345 and screen Confirmation Message. Repeats: Submit Button, untouched: {missing}"

	if title != expectedTitle {
		t.Errorf("got title %q, want %q", title, expectedTitle)
	}
	if body != expectedBody {
		t.Errorf("got body %q, want %q", body, expectedBody)
	}
}

func TestTemplate_EdgeCases(t *testing.T) {
	// Empty string value replaces placeholder with empty string
	tmpl, _ := NewPopupTemplate("Title", "Value: [{click}]")
	_, body := tmpl.Render(map[string]string{"click": ""})
	if body != "Value: []" {
		t.Errorf("got body %q, want 'Value: []'", body)
	}

	// No variables map (nil / empty)
	title, body := tmpl.Render(nil)
	if title != "Title" || body != "Value: [{click}]" {
		t.Errorf("expected untouched template for nil vars: %s / %s", title, body)
	}

	// Unicode and emojis
	tmplUnicode, _ := NewPopupTemplate("Уведомление: {window_title}", "Клик: {click} 🚀")
	uTitle, uBody := tmplUnicode.Render(map[string]string{
		"window_title": "Отчет.xlsx - Excel",
		"click":        "Удалить",
	})
	if uTitle != "Уведомление: Отчет.xlsx - Excel" || uBody != "Клик: Удалить 🚀" {
		t.Errorf("unicode render failed: %s / %s", uTitle, uBody)
	}
}
