// Package rules implements rule definition and matching logic.
package rules

// OCRSpecification matches any text recognized on the screen against pattern list.
type OCRSpecification struct {
	Patterns PatternList
}

// VariableKey returns the template variable name "ocr".
func (s OCRSpecification) VariableKey() string { return "ocr" }

// IsSatisfiedBy checks if any text on screen matches the pattern.
func (s OCRSpecification) IsSatisfiedBy(ctx RuleEvaluationContext) (string, bool) {
	if s.Patterns.IsEmpty() {
		return "", true
	}
	for _, text := range ctx.OCRScreenTexts {
		if s.Patterns.Matches(text) {
			return text, true
		}
	}
	return "", false
}
