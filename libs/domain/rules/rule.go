// Package rules implements rule definition and matching logic.
package rules

import (
	"errors"
	"strings"
)

// ErrEmptyRuleID is returned when creating a rule with an empty identifier.
var ErrEmptyRuleID = errors.New("rule id cannot be empty")

// WhenConditions holds optional pattern conditions.
// A rule fires when ALL defined (non-nil) conditions are satisfied.
type WhenConditions struct {
	Click       *PatternList
	Process     *PatternList
	WindowTitle *PatternList
	Clipboard   *PatternList
	OCR         *PatternList
}

// Rule represents a business rule that triggers a popup when conditions match.
type Rule struct {
	ID    string
	When  WhenConditions
	Popup PopupTemplate
}

// NewRule creates and validates a Rule entity.
func NewRule(id string, when WhenConditions, popup PopupTemplate) (Rule, error) {
	if strings.TrimSpace(id) == "" {
		return Rule{}, ErrEmptyRuleID
	}
	return Rule{
		ID:    id,
		When:  when,
		Popup: popup,
	}, nil
}

// RuleEvaluationContext holds the context against which a rule is evaluated.
type RuleEvaluationContext struct {
	ClickText      string
	Process        string
	WindowTitle    string
	ClipboardText  string
	OCRScreenTexts []string
}

// Evaluate checks if the rule matches the context. If matched, returns rendered title, body, and true.
func (r Rule) Evaluate(ctx RuleEvaluationContext) (string, string, bool) {
	vars := make(map[string]string)

	if !r.matchClick(ctx.ClickText, vars) {
		return "", "", false
	}
	if !r.matchProcess(ctx.Process, vars) {
		return "", "", false
	}
	if !r.matchWindowTitle(ctx.WindowTitle, vars) {
		return "", "", false
	}
	if !r.matchClipboard(ctx.ClipboardText, vars) {
		return "", "", false
	}
	if !r.matchOCR(ctx.OCRScreenTexts, vars) {
		return "", "", false
	}

	r.populateDefaults(ctx, vars)
	title, body := r.Popup.Render(vars)
	return title, body, true
}

func (r Rule) matchClick(clickText string, vars map[string]string) bool {
	if r.When.Click == nil || r.When.Click.IsEmpty() {
		return true
	}
	if clickText == "" || !r.When.Click.Matches(clickText) {
		return false
	}
	vars["click"] = clickText
	return true
}

func (r Rule) matchProcess(process string, vars map[string]string) bool {
	if r.When.Process == nil || r.When.Process.IsEmpty() {
		return true
	}
	if process == "" || !r.When.Process.Matches(process) {
		return false
	}
	vars["process"] = process
	return true
}

func (r Rule) matchWindowTitle(title string, vars map[string]string) bool {
	if r.When.WindowTitle == nil || r.When.WindowTitle.IsEmpty() {
		return true
	}
	if title == "" || !r.When.WindowTitle.Matches(title) {
		return false
	}
	vars["window_title"] = title
	return true
}

func (r Rule) matchClipboard(clipboard string, vars map[string]string) bool {
	if r.When.Clipboard == nil || r.When.Clipboard.IsEmpty() {
		return true
	}
	if clipboard == "" || !r.When.Clipboard.Matches(clipboard) {
		return false
	}
	vars["clipboard"] = clipboard
	return true
}

func (r Rule) matchOCR(screenTexts []string, vars map[string]string) bool {
	if r.When.OCR == nil || r.When.OCR.IsEmpty() {
		return true
	}
	for _, text := range screenTexts {
		if r.When.OCR.Matches(text) {
			vars["ocr"] = text
			return true
		}
	}
	return false
}

func (r Rule) populateDefaults(ctx RuleEvaluationContext, vars map[string]string) {
	if _, ok := vars["window_title"]; !ok && ctx.WindowTitle != "" {
		vars["window_title"] = ctx.WindowTitle
	}
	if _, ok := vars["click"]; !ok && ctx.ClickText != "" {
		vars["click"] = ctx.ClickText
	}
	if _, ok := vars["clipboard"]; !ok && ctx.ClipboardText != "" {
		vars["clipboard"] = ctx.ClipboardText
	}
	if _, ok := vars["process"]; !ok && ctx.Process != "" {
		vars["process"] = ctx.Process
	}
}
