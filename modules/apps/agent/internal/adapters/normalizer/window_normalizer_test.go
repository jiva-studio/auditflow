package normalizer

import (
	"testing"

	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/geometry"
	"assessment/modules/libs/domain/rules"
)

func TestWindowNormalizer_DefaultIgnoredTitles(t *testing.T) {
	norm := NewWindowNormalizer()

	initialCtx := rules.RuleEvaluationContext{
		Process:     "OUTLOOK.EXE",
		WindowTitle: "Confidential Email - Outlook",
		ClickText:   "Download",
	}

	// Case 1: Mouse click on Chrome Legacy Window placeholder should NOT overwrite process or title
	evChromeLegacy := events.MouseEvent{
		ProcessName: "msedgewebview2.exe",
		WindowTitle: "Chrome Legacy Window",
		Position:    geometry.Point{X: 100, Y: 100},
	}

	res := norm.NormalizeClick(initialCtx, evChromeLegacy)
	if res.Process != "OUTLOOK.EXE" {
		t.Errorf("expected process to remain OUTLOOK.EXE, got %q", res.Process)
	}
	if res.WindowTitle != "Confidential Email - Outlook" {
		t.Errorf("expected window title to remain Confidential Email - Outlook, got %q", res.WindowTitle)
	}
	if res.ClickText != "Download" {
		t.Errorf("expected click text to remain Download, got %q", res.ClickText)
	}

	// Case 2: Mouse click with valid distinct window title SHOULD overwrite
	evValid := events.MouseEvent{
		ProcessName: "notepad.exe",
		WindowTitle: "Untitled - Notepad",
	}

	resValid := norm.NormalizeClick(initialCtx, evValid)
	if resValid.Process != "notepad.exe" {
		t.Errorf("expected process to be notepad.exe, got %q", resValid.Process)
	}
	if resValid.WindowTitle != "Untitled - Notepad" {
		t.Errorf("expected window title to be Untitled - Notepad, got %q", resValid.WindowTitle)
	}
}

func TestWindowNormalizer_CustomIgnoredTitles(t *testing.T) {
	norm := NewWindowNormalizer("Custom Tooltip Overlay", "TrayPopup")

	initialCtx := rules.RuleEvaluationContext{
		Process:     "app.exe",
		WindowTitle: "Main App Window",
	}

	evIgnored := events.MouseEvent{
		ProcessName: "helper.exe",
		WindowTitle: "Custom Tooltip Overlay",
	}

	resIgnored := norm.NormalizeClick(initialCtx, evIgnored)
	if resIgnored.Process != "app.exe" || resIgnored.WindowTitle != "Main App Window" {
		t.Errorf("expected context to be preserved for ignored title, got %+v", resIgnored)
	}

	// Title that would have been ignored in default config, but is allowed now
	evDefault := events.MouseEvent{
		ProcessName: "msedgewebview2.exe",
		WindowTitle: "Chrome Legacy Window",
	}

	resDefault := norm.NormalizeClick(initialCtx, evDefault)
	if resDefault.Process != "msedgewebview2.exe" || resDefault.WindowTitle != "Chrome Legacy Window" {
		t.Errorf("expected Chrome Legacy Window to be accepted when not in custom ignored list, got %+v", resDefault)
	}
}

func TestWindowNormalizer_PartialOverrides(t *testing.T) {
	norm := NewWindowNormalizer()

	initialCtx := rules.RuleEvaluationContext{
		Process:     "app.exe",
		WindowTitle: "Main Window",
	}

	// Only process provided
	res1 := norm.NormalizeClick(initialCtx, events.MouseEvent{ProcessName: "new_app.exe"})
	if res1.Process != "new_app.exe" || res1.WindowTitle != "Main Window" {
		t.Errorf("expected only process to be overwritten, got %+v", res1)
	}

	// Only window title provided
	res2 := norm.NormalizeClick(initialCtx, events.MouseEvent{WindowTitle: "New Window Title"})
	if res2.Process != "app.exe" || res2.WindowTitle != "New Window Title" {
		t.Errorf("expected only window title to be overwritten, got %+v", res2)
	}
}
