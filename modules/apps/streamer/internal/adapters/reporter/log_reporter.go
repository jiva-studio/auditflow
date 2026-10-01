// Package reporter provides adapters for reporting errors and warnings.
package reporter

import (
	"fmt"
	"io"
	"os"
	"sync"

	"assessment/modules/apps/streamer/internal/ports"
)

// LogReporter implements ports.ErrorReporter writing structured text to an io.Writer.
type LogReporter struct {
	mu  sync.Mutex
	out io.Writer
}

// NewLogReporter creates a new LogReporter writing to the given writer (defaults to os.Stderr if nil).
func NewLogReporter(out io.Writer) *LogReporter {
	if out == nil {
		out = os.Stderr
	}
	return &LogReporter{out: out}
}

var _ ports.ErrorReporter = (*LogReporter)(nil)

// ReportError logs an error message with contextual fields.
func (r *LogReporter) ReportError(err error, context map[string]any) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.out, "[ERROR] %v context=%v\n", err, context)
}

// ReportWarning logs a warning message with contextual fields.
func (r *LogReporter) ReportWarning(msg string, context map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.out, "[WARN] %s context=%v\n", msg, context)
}
