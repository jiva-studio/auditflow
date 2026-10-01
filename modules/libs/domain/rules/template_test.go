package rules

import (
	"strings"
	"sync"
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
	expectedBody := "Clicked Submit Button with clip TOKEN-12345 and screen Confirmation Message. Repeats: Submit Button, untouched: "

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

	// No variables map (nil) -> strips unresolved placeholders
	title, body := tmpl.Render(nil)
	if title != "Title" || body != "Value: []" {
		t.Errorf("expected stripped placeholder for nil vars: %s / %s", title, body)
	}

	// Empty map -> strips unresolved placeholders
	title, body = tmpl.Render(map[string]string{})
	if title != "Title" || body != "Value: []" {
		t.Errorf("expected stripped placeholder for empty map: %s / %s", title, body)
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

func TestTemplate_UnresolvedPlaceholderStripping(t *testing.T) {
	// Multiple missing variables in title and body
	tmpl, err := NewPopupTemplate(
		"Alert {app} on {host}",
		"User {user} clicked {click} on {window_title} with {clipboard} and {ocr} info: {extra}",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	title, body := tmpl.Render(map[string]string{
		"user":  "Alice",
		"click": "Save",
	})

	expectedTitle := "Alert  on "
	expectedBody := "User Alice clicked Save on  with  and  info: "

	if title != expectedTitle {
		t.Errorf("got title %q, want %q", title, expectedTitle)
	}
	if body != expectedBody {
		t.Errorf("got body %q, want %q", body, expectedBody)
	}

	// Adjacent placeholders
	tmplAdj, _ := NewPopupTemplate("{a}{b}{c}", "prefix {x}{y}{z} suffix")
	adjTitle, adjBody := tmplAdj.Render(map[string]string{"b": "B", "y": "Y"})
	if adjTitle != "B" {
		t.Errorf("got adjTitle %q, want %q", adjTitle, "B")
	}
	if adjBody != "prefix Y suffix" {
		t.Errorf("got adjBody %q, want %q", adjBody, "prefix Y suffix")
	}
}

func TestTemplate_RecursiveInjectionDefense(t *testing.T) {
	tmpl, err := NewPopupTemplate("Alert {click}", "Copied: {clipboard}, Clicked: {click}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vars := map[string]string{
		"clipboard": "{clipboard}{click}{window_title}",
		"click":     "{clipboard}",
	}

	title, body := tmpl.Render(vars)
	if title != "Alert {clipboard}" {
		t.Errorf("got title %q, want %q", title, "Alert {clipboard}")
	}
	expectedBody := "Copied: {clipboard}{click}{window_title}, Clicked: {clipboard}"
	if body != expectedBody {
		t.Errorf("got body %q, want %q", body, expectedBody)
	}
}

func TestTemplate_MalformedTokens(t *testing.T) {
	tmpl, err := NewPopupTemplate("T: {{{click} and {unknown_var}", "B: { and } and {{}} and {999} and {   }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	title, body := tmpl.Render(map[string]string{
		"click": "CLICKED",
		"999":   "NUMBER",
	})

	expectedTitle := "T: {{CLICKED and "
	expectedBody := "B: { and } and {{}} and NUMBER and {   }"
	if title != expectedTitle {
		t.Errorf("got title %q, want %q", title, expectedTitle)
	}
	if body != expectedBody {
		t.Errorf("got body %q, want %q", body, expectedBody)
	}
}

func TestTemplate_UnicodeAndLargePayloads(t *testing.T) {
	tmpl, _ := NewPopupTemplate(
		"Alert \u200B \u202E 👩‍💻 {action}",
		"Body: \u200B {payload} \u202E {missing}",
	)

	largePayload := strings.Repeat("TORTURE-PAYLOAD-64KB-", 3200) // ~67KB
	title, body := tmpl.Render(map[string]string{
		"action":  "DELETED",
		"payload": largePayload,
	})

	expectedTitle := "Alert \u200B \u202E 👩‍💻 DELETED"
	if title != expectedTitle {
		t.Errorf("got title %q, want %q", title, expectedTitle)
	}
	if !strings.HasPrefix(body, "Body: \u200B TORTURE-PAYLOAD-64KB-") {
		t.Errorf("body missing large payload prefix")
	}
	if !strings.HasSuffix(body, " \u202E ") {
		t.Errorf("body missing stripped suffix: %q", body[len(body)-20:])
	}
}

func TestTemplate_ConcurrentRenders(t *testing.T) {
	tmpl, _ := NewPopupTemplate("Title {id}: {title_var}", "Body {id}: {body_var} - {missing}")
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vars := map[string]string{
				"id":        "123",
				"title_var": "TVar",
				"body_var":  "BVar",
			}
			title, body := tmpl.Render(vars)
			if title != "Title 123: TVar" {
				t.Errorf("concurrent title mismatch: %q", title)
			}
			if body != "Body 123: BVar - " {
				t.Errorf("concurrent body mismatch: %q", body)
			}
		}()
	}
	wg.Wait()
}
