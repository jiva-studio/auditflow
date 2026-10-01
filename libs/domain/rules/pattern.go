// Package rules implements rule definition and matching logic.
package rules

import (
	"strings"
)

// Pattern represents a single wildcard-enabled, case-insensitive string matcher without regex.
type Pattern struct {
	raw string
}

// NewPattern creates a string pattern where '*' matches any sequence of characters.
// Matching is case-insensitive and matches the whole text.
func NewPattern(raw string) (Pattern, error) {
	return Pattern{raw: raw}, nil
}

// Matches checks if the provided text satisfies the pattern.
func (p Pattern) Matches(text string) bool {
	return matchWildcard(strings.ToLower(p.raw), strings.ToLower(text))
}

// Raw returns the original pattern string.
func (p Pattern) Raw() string {
	return p.raw
}

// matchWildcard implements fast, non-allocating O(N+M) wildcard matching with '*'.
func matchWildcard(pattern, text string) bool {
	pIdx, tIdx := 0, 0
	starIdx, matchIdx := -1, 0

	for tIdx < len(text) {
		if pIdx < len(pattern) && (pattern[pIdx] == text[tIdx] || pattern[pIdx] == '?') {
			pIdx++
			tIdx++
			continue
		}
		if pIdx < len(pattern) && pattern[pIdx] == '*' {
			starIdx = pIdx
			matchIdx = tIdx
			pIdx++
			continue
		}
		if starIdx != -1 {
			pIdx = starIdx + 1
			matchIdx++
			tIdx = matchIdx
			continue
		}
		return false
	}

	for pIdx < len(pattern) && pattern[pIdx] == '*' {
		pIdx++
	}

	return pIdx == len(pattern)
}

// PatternList holds one or more patterns with OR semantics (matches if ANY pattern matches).
type PatternList struct {
	patterns []Pattern
}

// NewPatternList creates a PatternList from one or more raw pattern strings.
func NewPatternList(raws ...string) (PatternList, error) {
	patterns := make([]Pattern, 0, len(raws))
	for _, raw := range raws {
		p, err := NewPattern(raw)
		if err != nil {
			return PatternList{}, err
		}
		patterns = append(patterns, p)
	}
	return PatternList{patterns: patterns}, nil
}

// Matches returns true if any pattern in the list matches the text.
func (pl PatternList) Matches(text string) bool {
	if len(pl.patterns) == 0 {
		return false
	}
	for _, p := range pl.patterns {
		if p.Matches(text) {
			return true
		}
	}
	return false
}

// IsEmpty returns true if the list contains no patterns.
func (pl PatternList) IsEmpty() bool {
	return len(pl.patterns) == 0
}
