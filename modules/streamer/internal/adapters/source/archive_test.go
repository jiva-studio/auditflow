package source_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"assessment/modules/streamer/internal/adapters/reporter"
	"assessment/modules/streamer/internal/adapters/source"
)

func createTestTarGz(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write content: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test_recording.tar.gz")
	if err := os.WriteFile(tmpFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return tmpFile
}

func TestTarGzEventSource_LoadMetadata_Success(t *testing.T) {
	metaJSON := `{
		"schema_version": "1.0.0",
		"session_id": "sess-1",
		"employee_id": "emp-1",
		"started_at": "2026-03-10T10:00:00Z",
		"ended_at": "2026-03-10T10:00:10Z",
		"machine": {
			"hostname": "HOST-1",
			"os_version": "Windows 11",
			"displays": [
				{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}
			]
		}
	}`

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
	})
	src := source.NewTarGzEventSource(path, nil)

	meta, err := src.LoadMetadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.EmployeeID != "emp-1" {
		t.Errorf("got employee_id %q, want emp-1", meta.EmployeeID)
	}
	if len(meta.Displays) != 1 {
		t.Fatalf("got %d displays, want 1", len(meta.Displays))
	}
}

func TestTarGzEventSource_LoadMetadata_SchemaWarning(t *testing.T) {
	metaJSON := `{
		"schema_version": "2.0.0",
		"session_id": "sess-2",
		"employee_id": "emp-2",
		"started_at": "2026-03-10T10:00:00Z",
		"ended_at": "2026-03-10T10:00:10Z",
		"machine": {
			"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0}]
		}
	}`

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
	})
	mr := reporter.NewMockReporter()
	src := source.NewTarGzEventSource(path, mr)

	_, err := src.LoadMetadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mr.Items) != 1 || mr.Items[0].Level != "WARN" {
		t.Errorf("expected 1 WARN item in reporter, got %v", mr.Items)
	}
}

func TestTarGzEventSource_LoadMetadata_MissingOrCorrupted(t *testing.T) {
	t.Run("missing metadata", func(t *testing.T) {
		path := createTestTarGz(t, map[string]string{"events/other.txt": "hello"})
		src := source.NewTarGzEventSource(path, nil)
		if _, err := src.LoadMetadata(context.Background()); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		src := source.NewTarGzEventSource("/non/existent/path.tar.gz", nil)
		if _, err := src.LoadMetadata(context.Background()); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		path := createTestTarGz(t, map[string]string{"session/metadata.json": "{bad json"})
		src := source.NewTarGzEventSource(path, nil)
		if _, err := src.LoadMetadata(context.Background()); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("corrupt gzip archive", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "corrupt.tar.gz")
		_ = os.WriteFile(tmpFile, []byte("garbage not gzip"), 0644)
		src := source.NewTarGzEventSource(tmpFile, nil)
		if _, err := src.LoadMetadata(context.Background()); err == nil {
			t.Fatalf("expected error for corrupt gzip, got nil")
		}
	})
}

func TestTarGzEventSource_LoadMetadata_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"invalid time range", `{"session_id": "s1", "employee_id": "e1", "started_at": "2026-03-10T10:00:10Z", "ended_at": "2026-03-10T10:00:00Z", "machine": {"displays": [{"id": 0, "bounds": [0, 0, 100, 100], "scale": 1.0}]}}`},
		{"negative bounds", `{"session_id": "s1", "employee_id": "e1", "started_at": "2026-03-10T10:00:00Z", "ended_at": "2026-03-10T10:00:10Z", "machine": {"displays": [{"id": 0, "bounds": [0, 0, -100, 100], "scale": 1.0}]}}`},
		{"negative scale", `{"session_id": "s1", "employee_id": "e1", "started_at": "2026-03-10T10:00:00Z", "ended_at": "2026-03-10T10:00:10Z", "machine": {"displays": [{"id": 0, "bounds": [0, 0, 100, 100], "scale": -1.0}]}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := createTestTarGz(t, map[string]string{"session/metadata.json": tc.json})
			src := source.NewTarGzEventSource(path, nil)
			if _, err := src.LoadMetadata(context.Background()); err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func getSampleEventArchiveFiles() map[string]string {
	metaJSON := `{
		"schema_version": "1.0.0", "session_id": "sess-1", "employee_id": "emp-1",
		"started_at": "2026-03-10T10:00:00Z", "ended_at": "2026-03-10T10:00:05Z",
		"machine": {"hostname": "HOST-1", "os_version": "Windows 11", "displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}]}
	}`
	windowsJSONL := `{"ts": "2026-03-10T09:59:59.000Z", "event": "focus_change", "window_title": "Old", "process_name": "old.exe", "window_rect": [0, 0, 100, 100]}
{"ts": "2026-03-10T10:00:00.500Z", "event": "focus_change", "window_title": "Outlook", "process_name": "OUTLOOK.EXE", "application_name": "Outlook", "window_rect": [0, 0, 100, 100], "window_state": "normal", "url": "", "domain": "", "dwell_time_ms": 1000}`
	mouseJSONL := `{"ts": "2026-03-10T10:00:01.200Z", "event": "click", "button": "left", "mouse_x": 100, "mouse_y": 200, "click_count": "single", "window_title": "Outlook", "process_name": "OUTLOOK.EXE"}`
	clipboardJSONL := `{"ts": "2026-03-10T10:00:02.100Z", "event": "clipboard_change", "clipboard_content_text": "INV-1234", "clipboard_content_length": 8, "clipboard_source_app": "OUTLOOK.EXE", "clipboard_format": ["CF_TEXT"]}`
	ocrJSONL := `{"ts": "2026-03-10T10:00:00.100Z", "filename": "1.jpg", "display_id": 0, "resolution": [1920, 1080], "display_scale_factor": 1.0, "window_rect": [0, 0, 1920, 1080], "ocr_text_blocks": [{"text": "FW: Invoice", "bounding_box": [100, 200, 50, 20], "confidence": 0.99}], "deduplicated_from": ""}`

	return map[string]string{
		"session/metadata.json":      metaJSON,
		"session/windows.jsonl":      windowsJSONL,
		"session/mouse.jsonl":        mouseJSONL,
		"session/clipboard.jsonl":    clipboardJSONL,
		"session/ocr.jsonl":          ocrJSONL,
		"session/keyboard.jsonl":     `{"ts": "2026-03-10T10:00:03.100Z", "event": "key_press"}`,
		"session/keystrokes.jsonl":   `{"ts": "2026-03-10T10:00:03.200Z", "key": "A"}`,
		"session/mouse_scroll.jsonl": `{"ts": "2026-03-10T10:00:04.100Z", "delta": 120}`,
		"session/mouse_drag.jsonl":   `{"ts": "2026-03-10T10:00:04.200Z", "from": [0, 0], "to": [10, 10]}`,
		"session/unknown_file.jsonl": `{"ts": "2026-03-10T10:00:04.300Z", "other": true}`,
	}
}

func TestTarGzEventSource_StreamTicks_AllEventTypes(t *testing.T) {
	path := createTestTarGz(t, getSampleEventArchiveFiles())
	src := source.NewTarGzEventSource(path, nil)

	tickCh, errCh, err := src.StreamTicks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting stream: %v", err)
	}

	var receivedTicks []int
	var totalEvents int
	for tick := range tickCh {
		receivedTicks = append(receivedTicks, tick.TickIndex)
		totalEvents += tick.Len()
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if len(receivedTicks) != 5 {
		t.Errorf("got %d ticks, want 5", len(receivedTicks))
	}
	if totalEvents != 8 {
		t.Errorf("got %d total events, want 8", totalEvents)
	}
}

func TestTarGzEventSource_MalformedEventsReported(t *testing.T) {
	metaJSON := `{
		"session_id": "s1", "employee_id": "e1",
		"started_at": "2026-03-10T10:00:00Z", "ended_at": "2026-03-10T10:00:02Z",
		"machine": {"displays": [{"id": 0, "bounds": [0, 0, 100, 100], "scale": 1.0}]}
	}`

	cases := []struct {
		name     string
		filename string
		content  string
	}{
		{"bad window", "session/windows.jsonl", "{bad json\n"},
		{"bad mouse", "session/mouse.jsonl", "{bad json\n"},
		{"bad clipboard", "session/clipboard.jsonl", "{bad json\n"},
		{"bad ocr", "session/ocr.jsonl", "{bad json\n"},
		{"bad generic", "session/keyboard.jsonl", "{bad json\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := createTestTarGz(t, map[string]string{
				"session/metadata.json": metaJSON,
				tc.filename:             tc.content,
			})
			mr := reporter.NewMockReporter()
			src := source.NewTarGzEventSource(path, mr)
			tickCh, errCh, err := src.StreamTicks(context.Background())
			if err != nil {
				t.Fatalf("unexpected stream start error: %v", err)
			}
			for tick := range tickCh {
				_ = tick.TickIndex
			}
			if err := <-errCh; err != nil {
				t.Fatalf("unexpected stream error: %v", err)
			}
			if len(mr.Items) != 1 || mr.Items[0].Level != "WARN" {
				t.Errorf("expected 1 WARN for malformed record, got %v", mr.Items)
			}
		})
	}
}

func TestTarGzEventSource_ContextCancel(t *testing.T) {
	metaJSON := `{
		"session_id": "s1", "employee_id": "e1",
		"started_at": "2026-03-10T10:00:00Z", "ended_at": "2026-03-10T10:00:10Z",
		"machine": {"displays": [{"id": 0, "bounds": [0, 0, 100, 100], "scale": 1.0}]}
	}`

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
	})
	src := source.NewTarGzEventSource(path, nil)
	ctx, cancel := context.WithCancel(context.Background())

	tickCh, errCh, err := src.StreamTicks(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	<-tickCh
	cancel()

	for tick := range tickCh {
		_ = tick.TickIndex
	}

	err = <-errCh
	if err == nil {
		t.Fatalf("expected context cancelled error, got nil")
	}
}
