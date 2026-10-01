// Package rules implements rule definition and matching logic.
package rules

import (
	"errors"
	"strings"
)

// ErrEmptyRuleID is returned when creating a rule with an empty identifier.
var ErrEmptyRuleID = errors.New("rule id cannot be empty")

// Rule represents a business rule composed of specifications and a popup template.
type Rule struct {
	ID    string
	Specs []Specification
	Popup PopupTemplate
}

// NewRule creates and validates a Rule entity with standard WhenConditions.
func NewRule(id string, when WhenConditions, popup PopupTemplate) (Rule, error) {
	return NewRuleWithSpecs(id, when.ToSpecifications(), popup)
}

// NewRuleWithSpecs creates and validates a Rule entity with a custom list of specifications.
func NewRuleWithSpecs(id string, specs []Specification, popup PopupTemplate) (Rule, error) {
	if strings.TrimSpace(id) == "" {
		return Rule{}, ErrEmptyRuleID
	}
	return Rule{
		ID:    id,
		Specs: specs,
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

// Evaluate checks if all rule specifications match the context.
func (r Rule) Evaluate(ctx RuleEvaluationContext) (string, string, bool) {
	vars := make(map[string]string, len(r.Specs))

	for _, spec := range r.Specs {
		val, ok := spec.IsSatisfiedBy(ctx)
		if !ok {
			return "", "", false
		}
		if key := spec.VariableKey(); key != "" && val != "" {
			vars[key] = val
		}
	}

	r.populateDefaults(ctx, vars)
	title, body := r.Popup.Render(vars)
	return title, body, true
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

// RequiresClick returns true if any specification in the rule requires a click event.
func (r Rule) RequiresClick() bool {
	for _, spec := range r.Specs {
		if spec.VariableKey() == "click" {
			return true
		}
	}
	return false
}

// DependsOnWindow returns true if any specification in the rule depends on window or process context.
func (r Rule) DependsOnWindow() bool {
	for _, spec := range r.Specs {
		if spec.VariableKey() == "window_title" || spec.VariableKey() == "process" {
			return true
		}
	}
	return false
}

// DependsOnClipboard returns true if any specification in the rule depends on clipboard context.
func (r Rule) DependsOnClipboard() bool {
	for _, spec := range r.Specs {
		if spec.VariableKey() == "clipboard" {
			return true
		}
	}
	return false
}
