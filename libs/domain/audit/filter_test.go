package audit_test

import (
	"testing"

	"assessment/libs/domain/audit"
)

func TestFilter_Matches(t *testing.T) {
	popup, err := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title", "Body")
	if err != nil {
		t.Fatalf("unexpected error creating popup: %v", err)
	}

	tests := []struct {
		name     string
		filter   audit.Filter
		expected bool
	}{
		{
			name:     "empty filter matches all",
			filter:   audit.Filter{},
			expected: true,
		},
		{
			name:     "matching employee",
			filter:   audit.Filter{Employee: "emp-1"},
			expected: true,
		},
		{
			name:     "non-matching employee",
			filter:   audit.Filter{Employee: "emp-2"},
			expected: false,
		},
		{
			name:     "matching rule",
			filter:   audit.Filter{Rule: "rule-1"},
			expected: true,
		},
		{
			name:     "non-matching rule",
			filter:   audit.Filter{Rule: "rule-2"},
			expected: false,
		},
		{
			name:     "matching both employee and rule",
			filter:   audit.Filter{Employee: "emp-1", Rule: "rule-1"},
			expected: true,
		},
		{
			name:     "matching employee but wrong rule",
			filter:   audit.Filter{Employee: "emp-1", Rule: "rule-2"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(popup); got != tt.expected {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.expected)
			}
		})
	}
}
