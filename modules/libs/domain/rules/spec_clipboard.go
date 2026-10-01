// Package rules implements rule definition and matching logic.
package rules

// ClipboardSpecification matches the clipboard text against pattern list.
type ClipboardSpecification struct {
	Patterns PatternList
}

// VariableKey returns the template variable name "clipboard".
func (s ClipboardSpecification) VariableKey() string { return "clipboard" }

// IsSatisfiedBy checks if clipboard content matches the pattern.
func (s ClipboardSpecification) IsSatisfiedBy(ctx RuleEvaluationContext) (string, bool) {
	if s.Patterns.IsEmpty() {
		return "", true
	}
	if ctx.ClipboardText == "" || !s.Patterns.Matches(ctx.ClipboardText) {
		return "", false
	}
	return ctx.ClipboardText, true
}
