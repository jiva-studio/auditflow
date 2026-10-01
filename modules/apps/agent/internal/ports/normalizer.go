package ports

import (
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/rules"
)

// ContextNormalizer adjusts or enriches rule evaluation contexts based on event metadata and OS-level quirks.
type ContextNormalizer interface {
	NormalizeClick(ctx rules.RuleEvaluationContext, mouse events.MouseEvent) rules.RuleEvaluationContext
}
