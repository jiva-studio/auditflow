// Package rules implements rule definition and matching logic.
package rules

// WhenConditions holds optional pattern conditions.
type WhenConditions struct {
	Click       *PatternList
	Process     *PatternList
	WindowTitle *PatternList
	Clipboard   *PatternList
	OCR         *PatternList
}

// ToSpecifications converts WhenConditions into a slice of Specification instances.
func (w WhenConditions) ToSpecifications() []Specification {
	specs := make([]Specification, 0, 5)
	appendSpec := func(pl *PatternList, factory func(PatternList) Specification) {
		if pl != nil && !pl.IsEmpty() {
			specs = append(specs, factory(*pl))
		}
	}

	appendSpec(w.Click, func(p PatternList) Specification { return ClickSpecification{Patterns: p} })
	appendSpec(w.Process, func(p PatternList) Specification { return ProcessSpecification{Patterns: p} })
	appendSpec(w.WindowTitle, func(p PatternList) Specification { return WindowTitleSpecification{Patterns: p} })
	appendSpec(w.Clipboard, func(p PatternList) Specification { return ClipboardSpecification{Patterns: p} })
	appendSpec(w.OCR, func(p PatternList) Specification { return OCRSpecification{Patterns: p} })

	return specs
}
