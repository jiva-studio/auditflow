package rules

import (
	"testing"
)

func TestRuleEvaluation(t *testing.T) {
	clickPattern, _ := NewPatternList("FW:*")
	processPattern, _ := NewPatternList("OUTLOOK.EXE", "olk.exe")
	popupTmpl, _ := NewPopupTemplate("Forwarded email", "You opened a forwarded email: {click}")

	rule, err := NewRule("forwarded-email-opened", WhenConditions{
		Click:   &clickPattern,
		Process: &processPattern,
	}, popupTmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Positive evaluation
	ctx := RuleEvaluationContext{
		ClickText: "FW: Budget 2026",
		Process:   "OUTLOOK.EXE",
	}
	title, body, matched := rule.Evaluate(ctx)
	if !matched {
		t.Fatal("expected rule to match")
	}
	if title != "Forwarded email" || body != "You opened a forwarded email: FW: Budget 2026" {
		t.Fatalf("unexpected output: %s / %s", title, body)
	}

	// Negative evaluation (wrong process)
	ctxWrongProc := RuleEvaluationContext{
		ClickText: "FW: Budget 2026",
		Process:   "chrome.exe",
	}
	_, _, matched = rule.Evaluate(ctxWrongProc)
	if matched {
		t.Fatal("rule should not match wrong process")
	}
}

func TestRuleWithOCRAndWindowTitle(t *testing.T) {
	clickPattern, _ := NewPatternList("Delete")
	winPattern, _ := NewPatternList("*| Salesforce*")
	ocrPattern, _ := NewPatternList("Are you sure you want to delete*")
	popupTmpl, _ := NewPopupTemplate("Record deleted", "Deleted in Salesforce: {window_title}")

	rule, _ := NewRule("salesforce-record-deleted", WhenConditions{
		Click:       &clickPattern,
		WindowTitle: &winPattern,
		OCR:         &ocrPattern,
	}, popupTmpl)

	ctx := RuleEvaluationContext{
		ClickText:   "Delete",
		WindowTitle: "00482913 | Case | Salesforce - Google Chrome",
		OCRScreenTexts: []string{
			"Cancel",
			"Are you sure you want to delete this record?",
			"Save",
		},
	}

	title, body, matched := rule.Evaluate(ctx)
	if !matched {
		t.Fatal("expected salesforce rule to match")
	}
	if title != "Record deleted" || body != "Deleted in Salesforce: 00482913 | Case | Salesforce - Google Chrome" {
		t.Fatalf("unexpected output: %s / %s", title, body)
	}

	// Negative OCR mismatch
	ctxNoOCR := RuleEvaluationContext{
		ClickText:   "Delete",
		WindowTitle: "00482913 | Case | Salesforce - Google Chrome",
		OCRScreenTexts: []string{
			"Cancel",
			"Are you sure you want to save?",
		},
	}
	_, _, matched = rule.Evaluate(ctxNoOCR)
	if matched {
		t.Fatal("rule should not match when OCR condition is unmet")
	}
}

func TestRule_ClipboardAndValidation(t *testing.T) {
	popupTmpl, _ := NewPopupTemplate("Invoice attached", "Invoice: {clipboard}")

	// Validation: Empty ID
	_, err := NewRule("", WhenConditions{}, popupTmpl)
	if err != ErrEmptyRuleID {
		t.Fatalf("expected ErrEmptyRuleID, got %v", err)
	}

	// Clipboard rule
	clipPattern, _ := NewPatternList("INV-*")
	procPattern, _ := NewPatternList("OUTLOOK.EXE")
	rule, err := NewRule("invoice-ready-to-attach", WhenConditions{
		Clipboard: &clipPattern,
		Process:   &procPattern,
	}, popupTmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Positive evaluation
	ctx := RuleEvaluationContext{
		ClipboardText: "INV-98213",
		Process:       "OUTLOOK.EXE",
	}
	title, body, matched := rule.Evaluate(ctx)
	if !matched {
		t.Fatal("expected clipboard rule to match")
	}
	if title != "Invoice attached" || body != "Invoice: INV-98213" {
		t.Fatalf("unexpected output: %s / %s", title, body)
	}

	// Negative evaluation (wrong clipboard prefix)
	ctxWrongClip := RuleEvaluationContext{
		ClipboardText: "DOC-98213",
		Process:       "OUTLOOK.EXE",
	}
	_, _, matched = rule.Evaluate(ctxWrongClip)
	if matched {
		t.Fatal("rule should not match wrong clipboard text")
	}

	// Negative evaluation (empty click when click condition present)
	clickPattern, _ := NewPatternList("ClickMe")
	clickRule, _ := NewRule("click-rule", WhenConditions{
		Click: &clickPattern,
	}, popupTmpl)
	_, _, matched = clickRule.Evaluate(RuleEvaluationContext{})
	if matched {
		t.Fatal("click rule should not match empty click context")
	}
}
