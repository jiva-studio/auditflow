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

func TestTemplate_NoSecondaryExpansion(t *testing.T) {
	tmpl, err := NewPopupTemplate(
		"Title: {clipboard}",
		"Body: clip={clipboard}, click={click}, window={window_title}",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vars := map[string]string{
		"clipboard":    "{click} and {window_title}",
		"click":        "PAYLOAD_CLICK",
		"window_title": "PAYLOAD_WINDOW",
	}

	title, body := tmpl.Render(vars)
	expectedTitle := "Title: {click} and {window_title}"
	expectedBody := "Body: clip={click} and {window_title}, click=PAYLOAD_CLICK, window=PAYLOAD_WINDOW"

	if title != expectedTitle {
		t.Errorf("secondary expansion occurred in title!\ngot:  %q\nwant: %q", title, expectedTitle)
	}
	if body != expectedBody {
		t.Errorf("secondary expansion occurred in body!\ngot:  %q\nwant: %q", body, expectedBody)
	}
}

func TestTemplate_DeterminismUnderIteration(t *testing.T) {
	tmpl, err := NewPopupTemplate(
		"{a}-{b}-{c}-{d}",
		"{d}_{c}_{b}_{a}",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	vars := map[string]string{
		"a": "{b}",
		"b": "{c}",
		"c": "{d}",
		"d": "FINAL",
	}

	expectedTitle := "{b}-{c}-{d}-FINAL"
	expectedBody := "FINAL_{d}_{c}_{b}"

	for i := 0; i < 200; i++ {
		title, body := tmpl.Render(vars)
		if title != expectedTitle {
			t.Fatalf("iteration %d non-deterministic title: got %q, want %q", i, title, expectedTitle)
		}
		if body != expectedBody {
			t.Fatalf("iteration %d non-deterministic body: got %q, want %q", i, body, expectedBody)
		}
	}
}

func checkRender(t *testing.T, titleTmpl, bodyTmpl, expTitle, expBody string, vars map[string]string) {
	t.Helper()
	tmpl, err := NewPopupTemplate(titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("unexpected NewPopupTemplate error: %v", err)
	}
	title, body := tmpl.Render(vars)
	if title != expTitle {
		t.Errorf("title mismatch: got %q, want %q", title, expTitle)
	}
	if body != expBody {
		t.Errorf("body mismatch: got %q, want %q", body, expBody)
	}
}

func TestTemplate_Adversarial_Braces(t *testing.T) {
	// Self-referencing variable
	checkRender(t, "Self: {self}", "Body: {self}", "Self: {self}", "Body: {self}", map[string]string{"self": "{self}"})

	// Giant adjacent tokens
	vars := map[string]string{"click": "1", "window_title": "2", "clipboard": "3", "ocr": "4"}
	checkRender(t, "{click}{click}{window_title}{clipboard}{ocr}", "{ocr}{clipboard}{window_title}{click}{click}", "11234", "43211", vars)

	// Nested braces in template
	checkRender(t, "{{click}}", "prefix_{recursive_{click}}_suffix", "{OK}", "prefix_{recursive_OK}_suffix", map[string]string{"click": "OK"})

	// Unclosed braces and stray closing braces
	checkRender(t, "Unclosed {{{ and {valid} after", "Closing only }}} and {valid} then {", "Unclosed {{{ and REPLACED after", "Closing only }}} and REPLACED then {", map[string]string{"valid": "REPLACED"})

	// Empty braces
	checkRender(t, "Empty {} and with space { }", "Normal {click}", "Empty EMPTY_VAL and with space { }", "Normal CLICK_VAL", map[string]string{"": "EMPTY_VAL", "click": "CLICK_VAL"})
}

func TestTemplate_Adversarial_UnicodeAndControlChars(t *testing.T) {
	// Non-ASCII UTF-8, emojis, zero-width spaces
	uVars := map[string]string{
		"user": "Иван",
		"msg":  "Всё работает 🚀",
		"lang": "العربية",
	}
	checkRender(t, "Привет 🌟 {user}\u200B!", "Текст: {msg} — 中文 / 日本語 / العربية: {lang}", "Привет 🌟 Иван\u200B!", "Текст: Всё работает 🚀 — 中文 / 日本語 / العربية: العربية", uVars)

	// Control characters and newlines in variables
	cVars := map[string]string{
		"data": "line1\nline2\r\n\t\x00test",
	}
	checkRender(t, "Title with newline: {data}", "Body: \t{data}\n\r", "Title with newline: line1\nline2\r\n\t\x00test", "Body: \tline1\nline2\r\n\t\x00test\n\r", cVars)
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

func BenchmarkTemplate_Render(b *testing.B) {
	tmpl, _ := NewPopupTemplate(
		"Action on {window_title} ({process})",
		"Clicked {click} with clip {clipboard} and screen {ocr}. Repeats: {click}, untouched: {missing}",
	)
	vars := map[string]string{
		"click":        "Submit Button",
		"process":      "chrome.exe",
		"window_title": "Admin Dashboard",
		"clipboard":    "{click} injection payload",
		"ocr":          "Confirmation Message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Render(vars)
	}
}
