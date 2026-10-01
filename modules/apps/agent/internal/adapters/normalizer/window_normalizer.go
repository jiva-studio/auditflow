// Package normalizer provides strategy implementations for sanitizing and enriching evaluation contexts.
package normalizer

import (
	"assessment/modules/apps/agent/internal/ports"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/rules"
)

// WindowNormalizer filters out transient or internal placeholder window titles (such as Chromium render widgets).
type WindowNormalizer struct {
	ignoredTitles map[string]struct{}
}

// DefaultIgnoredTitles lists known internal child window titles that should not overwrite active top-level contexts.
var DefaultIgnoredTitles = []string{
	"Chrome Legacy Window",
}

// NewWindowNormalizer creates a new WindowNormalizer with the given ignored window titles.
// If no titles are provided, DefaultIgnoredTitles are used.
func NewWindowNormalizer(ignoredTitles ...string) *WindowNormalizer {
	if len(ignoredTitles) == 0 {
		ignoredTitles = DefaultIgnoredTitles
	}
	ignored := make(map[string]struct{}, len(ignoredTitles))
	for _, t := range ignoredTitles {
		if t != "" {
			ignored[t] = struct{}{}
		}
	}
	return &WindowNormalizer{ignoredTitles: ignored}
}

var _ ports.ContextNormalizer = (*WindowNormalizer)(nil)

// NormalizeClick applies event process name and window title if the window title is not in the ignored set.
func (n *WindowNormalizer) NormalizeClick(ctx rules.RuleEvaluationContext, m events.MouseEvent) rules.RuleEvaluationContext {
	if _, ignored := n.ignoredTitles[m.WindowTitle]; ignored {
		return ctx
	}
	if m.ProcessName != "" {
		ctx.Process = m.ProcessName
	}
	if m.WindowTitle != "" {
		ctx.WindowTitle = m.WindowTitle
	}
	return ctx
}
