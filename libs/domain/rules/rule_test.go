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
}
