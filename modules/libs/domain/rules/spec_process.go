// Package rules implements rule definition and matching logic.
package rules

// ProcessSpecification matches the process name against pattern list.
type ProcessSpecification struct {
	Patterns PatternList
}

// VariableKey returns the template variable name "process".
func (s ProcessSpecification) VariableKey() string { return "process" }

// IsSatisfiedBy checks if active process matches the pattern.
func (s ProcessSpecification) IsSatisfiedBy(ctx RuleEvaluationContext) (string, bool) {
	if s.Patterns.IsEmpty() {
		return "", true
	}
	if ctx.Process == "" || !s.Patterns.Matches(ctx.Process) {
		return "", false
	}
	return ctx.Process, true
}
