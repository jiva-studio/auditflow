package reporter_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"assessment/modules/streamer/internal/adapters/reporter"
)

func TestLogReporter(t *testing.T) {
	var buf bytes.Buffer
	rep := reporter.NewLogReporter(&buf)

	rep.ReportError(errors.New("something went wrong"), map[string]any{"line": 42})
	rep.ReportWarning("deprecated field used", map[string]any{"field": "old_name"})
	rep.ReportError(nil, nil)

	out := buf.String()
	if !strings.Contains(out, "[ERROR] something went wrong") {
		t.Errorf("expected ERROR log, got %s", out)
	}
	if !strings.Contains(out, "[WARN] deprecated field used") {
		t.Errorf("expected WARN log, got %s", out)
	}

	defaultRep := reporter.NewLogReporter(nil)
	if defaultRep == nil {
		t.Fatalf("expected non-nil default reporter")
	}
}

func TestMockReporter(t *testing.T) {
	mr := reporter.NewMockReporter()
	mr.ReportError(errors.New("mock error"), map[string]any{"file": "a.json"})
	mr.ReportWarning("mock warning", map[string]any{"file": "b.json"})
	mr.ReportError(nil, nil)

	if len(mr.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(mr.Items))
	}
	if mr.Items[0].Level != "ERROR" {
		t.Errorf("got level %s, want ERROR", mr.Items[0].Level)
	}
	if mr.Items[1].Level != "WARN" {
		t.Errorf("got level %s, want WARN", mr.Items[1].Level)
	}

	mr.Clear()
	if len(mr.Items) != 0 {
		t.Errorf("expected 0 items after Clear, got %d", len(mr.Items))
	}
}
