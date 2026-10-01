// Package rules implements rule definition and matching logic.
package rules

// ClickSpecification matches the click text against pattern list.
type ClickSpecification struct {
	Patterns PatternList
}

// VariableKey returns the template variable name "click".
func (s ClickSpecification) VariableKey() string { return "click" }

// IsSatisfiedBy checks if click text matches the pattern.
func (s ClickSpecification) IsSatisfiedBy(ctx RuleEvaluationContext) (string, bool) {
	if s.Patterns.IsEmpty() {
		return "", true
	}
	if ctx.ClickText == "" || !s.Patterns.Matches(ctx.ClickText) {
		return "", false
	}
	return ctx.ClickText, true
}
