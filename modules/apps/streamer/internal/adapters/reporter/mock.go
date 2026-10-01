// Package reporter provides adapters for reporting errors and warnings.
package reporter

import (
	"sync"

	"assessment/modules/apps/streamer/internal/ports"
)

// ReportedItem captures a single error or warning report.
type ReportedItem struct {
	Level   string
	Message string
	Error   error
	Context map[string]any
}

// MockReporter is an in-memory ErrorReporter for test assertions.
type MockReporter struct {
	mu    sync.Mutex
	Items []ReportedItem
}

// NewMockReporter constructs a new MockReporter.
func NewMockReporter() *MockReporter {
	return &MockReporter{
		Items: make([]ReportedItem, 0),
	}
}

var _ ports.ErrorReporter = (*MockReporter)(nil)

// ReportError records an error.
func (m *MockReporter) ReportError(err error, context map[string]any) {
	if err == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Items = append(m.Items, ReportedItem{
		Level:   "ERROR",
		Error:   err,
		Context: context,
	})
}

// ReportWarning records a warning.
func (m *MockReporter) ReportWarning(msg string, context map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Items = append(m.Items, ReportedItem{
		Level:   "WARN",
		Message: msg,
		Context: context,
	})
}

// Clear resets recorded items.
func (m *MockReporter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Items = m.Items[:0]
}
