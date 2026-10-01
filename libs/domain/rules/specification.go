// Package rules implements rule definition and matching logic.
package rules

// Specification defines an extensible, strongly-typed condition against an evaluation context.
type Specification interface {
	// VariableKey returns the placeholder name for template rendering (e.g., "click", "process").
	VariableKey() string
	// IsSatisfiedBy evaluates the condition against the context and returns the matched text value.
	IsSatisfiedBy(ctx RuleEvaluationContext) (matchedValue string, ok bool)
}
