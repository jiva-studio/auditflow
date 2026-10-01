// Package rules provides JSON loading, validation, and conversion into domain rules.
package rules

import (
	"bytes"
	"encoding/json"
	"errors"
)

// ErrInvalidStringOrSlice is returned when JSON is neither a string, an array of strings, nor null.
var ErrInvalidStringOrSlice = errors.New("expected string, array of strings, or null")

// StringOrSlice deserializes a JSON field that can either be a string, a slice of strings, or null.
type StringOrSlice []string

// UnmarshalJSON implements json.Unmarshaler.
func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*s = nil
		return nil
	}

	if trimmed[0] == '"' {
		var single string
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return err
		}
		*s = []string{single}
		return nil
	}

	if trimmed[0] == '[' {
		var slice []string
		if err := json.Unmarshal(trimmed, &slice); err != nil {
			return err
		}
		*s = slice
		return nil
	}

	return ErrInvalidStringOrSlice
}
