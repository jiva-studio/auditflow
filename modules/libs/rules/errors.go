// Package rules provides JSON loading, validation, and conversion into domain rules.
package rules

import "errors"

var (
	// ErrEmptyRuleID is returned when a rule definition has an empty ID.
	ErrEmptyRuleID = errors.New("rule id cannot be empty")
	// ErrMissingPopupTitle is returned when a rule popup has an empty or whitespace title.
	ErrMissingPopupTitle = errors.New("popup title cannot be empty")
	// ErrNilRule is returned when a rule definition in the array is null or empty.
	ErrNilRule = errors.New("rule cannot be nil")
)
