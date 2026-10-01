// Package rules implements rule definition and matching logic.
package rules

// WindowTitleSpecification matches the window title against pattern list.
type WindowTitleSpecification struct {
	Patterns PatternList
}

// VariableKey returns the template variable name "window_title".
func (s WindowTitleSpecification) VariableKey() string { return "window_title" }

// IsSatisfiedBy checks if active window title matches the pattern.
func (s WindowTitleSpecification) IsSatisfiedBy(ctx RuleEvaluationContext) (string, bool) {
	if s.Patterns.IsEmpty() {
		return "", true
	}
	if ctx.WindowTitle == "" || !s.Patterns.Matches(ctx.WindowTitle) {
		return "", false
	}
	return ctx.WindowTitle, true
}
