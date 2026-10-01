package rules

import "testing"

func TestClickSpecification(t *testing.T) {
	clickPatterns, _ := NewPatternList("Save", "Apply*")
	clickSpec := ClickSpecification{Patterns: clickPatterns}
	if clickSpec.VariableKey() != "click" {
		t.Errorf("expected key click, got %s", clickSpec.VariableKey())
	}

	val, ok := clickSpec.IsSatisfiedBy(RuleEvaluationContext{ClickText: "Apply Changes"})
	if !ok || val != "Apply Changes" {
		t.Errorf("expected click match, got %v, %s", ok, val)
	}

	emptyClickSpec := ClickSpecification{}
	if _, emptyOk := emptyClickSpec.IsSatisfiedBy(RuleEvaluationContext{}); !emptyOk {
		t.Error("expected empty click spec to satisfy")
	}
}

func TestProcessSpecification(t *testing.T) {
	procPatterns, _ := NewPatternList("chrome.exe")
	procSpec := ProcessSpecification{Patterns: procPatterns}
	if procSpec.VariableKey() != "process" {
		t.Errorf("expected key process, got %s", procSpec.VariableKey())
	}
	val, ok := procSpec.IsSatisfiedBy(RuleEvaluationContext{Process: "chrome.exe"})
	if !ok || val != "chrome.exe" {
		t.Errorf("expected process match, got %v, %s", ok, val)
	}
	emptyProcSpec := ProcessSpecification{}
	if _, emptyOk := emptyProcSpec.IsSatisfiedBy(RuleEvaluationContext{}); !emptyOk {
		t.Error("expected empty proc spec to satisfy")
	}
}

func TestWindowTitleSpecification(t *testing.T) {
	winPatterns, _ := NewPatternList("*Dashboard*")
	winSpec := WindowTitleSpecification{Patterns: winPatterns}
	if winSpec.VariableKey() != "window_title" {
		t.Errorf("expected key window_title, got %s", winSpec.VariableKey())
	}
	val, ok := winSpec.IsSatisfiedBy(RuleEvaluationContext{WindowTitle: "Admin Dashboard"})
	if !ok || val != "Admin Dashboard" {
		t.Errorf("expected window title match, got %v, %s", ok, val)
	}
	emptyWinSpec := WindowTitleSpecification{}
	if _, emptyOk := emptyWinSpec.IsSatisfiedBy(RuleEvaluationContext{}); !emptyOk {
		t.Error("expected empty win spec to satisfy")
	}
}

func TestClipboardSpecification(t *testing.T) {
	clipPatterns, _ := NewPatternList("SEC-*")
	clipSpec := ClipboardSpecification{Patterns: clipPatterns}
	if clipSpec.VariableKey() != "clipboard" {
		t.Errorf("expected key clipboard, got %s", clipSpec.VariableKey())
	}
	val, ok := clipSpec.IsSatisfiedBy(RuleEvaluationContext{ClipboardText: "SEC-999"})
	if !ok || val != "SEC-999" {
		t.Errorf("expected clipboard match, got %v, %s", ok, val)
	}
	emptyClipSpec := ClipboardSpecification{}
	if _, emptyOk := emptyClipSpec.IsSatisfiedBy(RuleEvaluationContext{}); !emptyOk {
		t.Error("expected empty clip spec to satisfy")
	}
}

func TestOCRSpecification(t *testing.T) {
	ocrPatterns, _ := NewPatternList("CONFIDENTIAL*")
	ocrSpec := OCRSpecification{Patterns: ocrPatterns}
	if ocrSpec.VariableKey() != "ocr" {
		t.Errorf("expected key ocr, got %s", ocrSpec.VariableKey())
	}
	val, ok := ocrSpec.IsSatisfiedBy(RuleEvaluationContext{OCRScreenTexts: []string{"Header", "CONFIDENTIAL DATA"}})
	if !ok || val != "CONFIDENTIAL DATA" {
		t.Errorf("expected ocr match, got %v, %s", ok, val)
	}
	emptyOCRSpec := OCRSpecification{}
	if _, emptyOk := emptyOCRSpec.IsSatisfiedBy(RuleEvaluationContext{}); !emptyOk {
		t.Error("expected empty ocr spec to satisfy")
	}
}
