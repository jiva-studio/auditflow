package source_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/reporter"
	"assessment/modules/apps/streamer/internal/adapters/source"
	"assessment/modules/libs/domain/events"
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
	if totalEvents != 9 {
		t.Errorf("got %d total events, want 9", totalEvents)
	}
}

func TestTarGzEventSource_PreRollInitialization(t *testing.T) {
	metaJSON := `{
		"schema_version": "1.0.0",
		"session_id": "sess-preroll",
		"employee_id": "emp-preroll",
		"started_at": "2026-03-10T10:00:00Z",
		"ended_at": "2026-03-10T10:00:02Z",
		"machine": {
			"hostname": "HOST-PREROLL",
			"os_version": "Windows 11",
			"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}]
		}
	}`

	windowsJSONL := `{"ts": "2026-03-10T09:59:58.000Z", "event": "focus_change", "window_title": "Desktop Start", "process_name": "explorer.exe", "window_rect": [0, 0, 1920, 1080]}
{"ts": "2026-03-10T10:00:00.500Z", "event": "focus_change", "window_title": "Outlook", "process_name": "OUTLOOK.EXE", "window_rect": [0, 0, 1920, 1080]}`

	ocrJSONL := `{"ts": "2026-03-10T09:59:59.000Z", "filename": "init.jpg", "display_id": 0, "resolution": [1920, 1080], "display_scale_factor": 1.0, "window_rect": [0, 0, 1920, 1080], "ocr_text_blocks": [{"text": "Welcome", "bounding_box": [0, 0, 100, 50], "confidence": 0.99}], "deduplicated_from": ""}`

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
		"session/windows.jsonl": windowsJSONL,
		"session/ocr.jsonl":     ocrJSONL,
	})

	src := source.NewTarGzEventSource(path, nil)
	tickCh, errCh, err := src.StreamTicks(context.Background())
	if err != nil {
		t.Fatalf("unexpected stream start error: %v", err)
	}

	var batches []events.TickBatch
	for tick := range tickCh {
		batches = append(batches, tick)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if len(batches) != 2 {
		t.Fatalf("got %d batches, want 2", len(batches))
	}

	tick0 := batches[0]
	if tick0.Len() != 3 {
		t.Fatalf("tick 0 length got %d, want 3", tick0.Len())
	}

	evs := tick0.Events()
	if evs[0].GetType() != events.EventTypeWindow {
		t.Errorf("expected first event to be WindowEvent, got %v", evs[0].GetType())
	}
	if evs[1].GetType() != events.EventTypeOCR {
		t.Errorf("expected second event to be OCREvent, got %v", evs[1].GetType())
	}
	if evs[2].GetType() != events.EventTypeWindow {
		t.Errorf("expected third event to be WindowEvent, got %v", evs[2].GetType())
	}
}

func TestTarGzEventSource_TrailingSessionEvents(t *testing.T) {
	metaJSON := `{
		"schema_version": "1.0.0",
		"session_id": "sess-trailing",
		"employee_id": "emp-trailing",
		"started_at": "2026-03-10T10:00:00Z",
		"ended_at": "2026-03-10T10:00:02Z",
		"machine": {
			"hostname": "HOST-1",
			"os_version": "Windows 11",
			"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}]
		}
	}`

	windowsJSONL := `{"ts": "2026-03-10T10:00:00.500Z", "event": "focus_change", "window_title": "Outlook", "process_name": "OUTLOOK.EXE", "window_rect": [0, 0, 1920, 1080]}
{"ts": "2026-03-10T10:00:02.000Z", "event": "focus_change", "window_title": "Trailing 1", "process_name": "app.exe", "window_rect": [0, 0, 1920, 1080]}
{"ts": "2026-03-10T10:00:02.500Z", "event": "focus_change", "window_title": "Trailing 2", "process_name": "app.exe", "window_rect": [0, 0, 1920, 1080]}`

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
		"session/windows.jsonl": windowsJSONL,
	})

	src := source.NewTarGzEventSource(path, nil)
	tickCh, errCh, err := src.StreamTicks(context.Background())
	if err != nil {
		t.Fatalf("unexpected stream start error: %v", err)
	}

	var batches []events.TickBatch
	for tick := range tickCh {
		batches = append(batches, tick)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if len(batches) != 3 {
		t.Fatalf("got %d batches, want 3", len(batches))
	}

	finalBatch := batches[2]
	if finalBatch.TickIndex != 2 {
		t.Errorf("got final tick index %d, want 2", finalBatch.TickIndex)
	}
	if finalBatch.Len() != 2 {
		t.Fatalf("got %d events in final batch, want 2", finalBatch.Len())
	}
}

func TestTarGzEventSource_BoundaryTimestamps(t *testing.T) {
	metaJSON := `{
		"schema_version": "1.0.0",
		"session_id": "sess-boundary",
		"employee_id": "emp-boundary",
		"started_at": "2026-03-10T10:00:00Z",
		"ended_at": "2026-03-10T10:00:02Z",
		"machine": {
			"hostname": "HOST-1",
			"os_version": "Windows 11",
			"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}]
		}
	}`

	windowsJSONL := `{"ts": "2026-03-10T10:00:00.000Z", "event": "focus_change", "window_title": "Tick 0 Exact Start", "process_name": "app.exe", "window_rect": [0, 0, 100, 100]}
{"ts": "2026-03-10T10:00:01.000Z", "event": "focus_change", "window_title": "Tick 1 Exact Start", "process_name": "app.exe", "window_rect": [0, 0, 100, 100]}`

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
		"session/windows.jsonl": windowsJSONL,
	})

	src := source.NewTarGzEventSource(path, nil)
	tickCh, errCh, err := src.StreamTicks(context.Background())
	if err != nil {
		t.Fatalf("unexpected stream start error: %v", err)
	}

	var batches []events.TickBatch
	for tick := range tickCh {
		batches = append(batches, tick)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if len(batches) != 2 {
		t.Fatalf("got %d batches, want 2", len(batches))
	}

	if batches[0].Len() != 1 || batches[0].Events()[0].(events.WindowEvent).WindowTitle != "Tick 0 Exact Start" {
		t.Errorf("unexpected batch 0: %+v", batches[0])
	}
	if batches[1].Len() != 1 || batches[1].Events()[0].(events.WindowEvent).WindowTitle != "Tick 1 Exact Start" {
		t.Errorf("unexpected batch 1: %+v", batches[1])
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

func writeTarHeaderAndData(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func generateSyntheticArchive(targetPath string, numSeconds int, eventsPerSec int) error {
	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	metaJSON := fmt.Sprintf(`{
		"schema_version": "1.0.0",
		"session_id": "sess-stress",
		"employee_id": "emp-stress",
		"started_at": "2026-03-10T10:00:00.000Z",
		"ended_at": "%s",
		"machine": {"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0}]}
	}`, time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC).Add(time.Duration(numSeconds)*time.Second).Format(time.RFC3339Nano))

	if err := writeTarHeaderAndData(tw, "session/metadata.json", []byte(metaJSON)); err != nil {
		return err
	}

	var winBuf, mouseBuf bytes.Buffer
	baseTime := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)

	for s := 0; s < numSeconds; s++ {
		for e := 0; e < eventsPerSec/2; e++ {
			ts := baseTime.Add(time.Duration(s)*time.Second + time.Duration(e*2)*time.Millisecond)
			tsStr := ts.Format(time.RFC3339Nano)
			fmt.Fprintf(&winBuf, `{"ts":"%s","event":"focus_change","window_title":"W%d","process_name":"app.exe","window_rect":[0,0,100,100]}`+"\n", tsStr, s)
			fmt.Fprintf(&mouseBuf, `{"ts":"%s","event":"click","button":"left","mouse_x":%d,"mouse_y":%d}`+"\n", tsStr, s, e)
		}
	}

	if err := writeTarHeaderAndData(tw, "session/windows.jsonl", winBuf.Bytes()); err != nil {
		return err
	}
	if err := writeTarHeaderAndData(tw, "session/mouse.jsonl", mouseBuf.Bytes()); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gzw.Close()
}

func TestTarGzEventSource_LargeDataset_MemoryBounded(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "stress_recording.tar.gz")
	const numSeconds = 200
	const eventsPerSec = 200 // 40,000 events total

	if err := generateSyntheticArchive(tmpFile, numSeconds, eventsPerSec); err != nil {
		t.Fatalf("failed to generate synthetic archive: %v", err)
	}

	runtime.GC()
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	src := source.NewTarGzEventSource(tmpFile, nil)
	tickCh, errCh, err := src.StreamTicks(context.Background())
	if err != nil {
		t.Fatalf("failed to start streaming: %v", err)
	}

	ticksReceived := 0
	eventsReceived := 0
	for tick := range tickCh {
		ticksReceived++
		eventsReceived += tick.Len()
	}

	if err := <-errCh; err != nil {
		t.Fatalf("stream returned error: %v", err)
	}

	if ticksReceived != numSeconds {
		t.Errorf("got %d ticks, want %d", ticksReceived, numSeconds)
	}
	if eventsReceived != numSeconds*eventsPerSec {
		t.Errorf("got %d events, want %d", eventsReceived, numSeconds*eventsPerSec)
	}

	runtime.GC()
	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)

	// HeapAlloc difference should remain small (well under 20MB)
	allocDeltaMB := float64(mAfter.HeapAlloc-mBefore.HeapAlloc) / (1024 * 1024)
	t.Logf("Stress test completed: %d ticks, %d events, HeapAlloc delta: %.2f MB", ticksReceived, eventsReceived, allocDeltaMB)
}

func TestTarGzEventSource_IdenticalTimestampsTieBreaker(t *testing.T) {
	metaJSON := `{
		"schema_version": "1.0.0",
		"session_id": "sess-tie",
		"employee_id": "emp-tie",
		"started_at": "2026-03-10T10:00:00.000Z",
		"ended_at": "2026-03-10T10:00:01.000Z",
		"machine": {"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0}]}
	}`

	// Exactly identical timestamps in different streams
	const sameTS = "2026-03-10T10:00:00.500Z"
	winJSONL := fmt.Sprintf(`{"ts": "%s", "event": "focus_change", "window_title": "W1", "process_name": "app.exe", "window_rect": [0, 0, 100, 100]}`+"\n", sameTS)
	mouseJSONL := fmt.Sprintf(`{"ts": "%s", "event": "click", "button": "left", "mouse_x": 10, "mouse_y": 20}`+"\n", sameTS)
	clipJSONL := fmt.Sprintf(`{"ts": "%s", "event": "clipboard_change", "clipboard_content_text": "text", "clipboard_content_length": 4}`+"\n", sameTS)

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json":   metaJSON,
		"session/windows.jsonl":   winJSONL,
		"session/mouse.jsonl":     mouseJSONL,
		"session/clipboard.jsonl": clipJSONL,
	})

	src := source.NewTarGzEventSource(path, nil)
	tickCh, errCh, err := src.StreamTicks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting stream: %v", err)
	}

	var totalEvents int
	for tick := range tickCh {
		totalEvents += tick.Len()
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if totalEvents != 3 {
		t.Errorf("got %d events, want 3", totalEvents)
	}
}

func TestTarGzEventSource_MetadataCachingAndSinglePassStream(t *testing.T) {
	metaJSON := `{
		"schema_version": "1.0.0",
		"session_id": "sess-cached",
		"employee_id": "emp-cached",
		"started_at": "2026-03-10T10:00:00.000Z",
		"ended_at": "2026-03-10T10:00:01.000Z",
		"machine": {"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0}]}
	}`
	winJSONL := `{"ts": "2026-03-10T10:00:00.500Z", "event": "focus_change", "window_title": "W1", "process_name": "app.exe", "window_rect": [0, 0, 100, 100]}` + "\n"

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
		"session/windows.jsonl": winJSONL,
	})

	src := source.NewTarGzEventSource(path, nil)

	// 1. Initial LoadMetadata
	meta1, err := src.LoadMetadata(context.Background())
	if err != nil {
		t.Fatalf("first LoadMetadata failed: %v", err)
	}
	if meta1.EmployeeID != "emp-cached" {
		t.Fatalf("expected emp-cached, got %s", meta1.EmployeeID)
	}

	// 2. Remove the underlying archive file to prove caching works without re-opening disk file
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove test file: %v", err)
	}

	// 3. Second LoadMetadata should return from memory cache without error
	meta2, err := src.LoadMetadata(context.Background())
	if err != nil {
		t.Fatalf("second LoadMetadata should succeed from cache, got error: %v", err)
	}
	if meta2.EmployeeID != meta1.EmployeeID {
		t.Errorf("got %s, want %s", meta2.EmployeeID, meta1.EmployeeID)
	}
}

func TestTarGzEventSource_InvalidGeometryWarnings(t *testing.T) {
	metaJSON := `{
		"schema_version": "1.0.0",
		"session_id": "sess-geom-warn",
		"employee_id": "emp-geom-warn",
		"started_at": "2026-03-10T10:00:00.000Z",
		"ended_at": "2026-03-10T10:00:01.000Z",
		"machine": {"displays": [{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0}]}
	}`
	// Negative width in window_rect and negative height in ocr bounding_box
	winJSONL := `{"ts": "2026-03-10T10:00:00.200Z", "event": "focus_change", "window_title": "W1", "process_name": "app.exe", "window_rect": [0, 0, -50, 100]}` + "\n"
	ocrJSONL := `{"ts": "2026-03-10T10:00:00.400Z", "filename": "s.jpg", "display_id": 0, "resolution": [1920, 1080], "window_rect": [0, 0, -100, 100], "ocr_text_blocks": [{"text": "Sample", "bounding_box": [10, 10, 50, -20], "confidence": 0.9}]}` + "\n"

	path := createTestTarGz(t, map[string]string{
		"session/metadata.json": metaJSON,
		"session/windows.jsonl": winJSONL,
		"session/ocr.jsonl":     ocrJSONL,
	})

	rep := reporter.NewMockReporter()
	src := source.NewTarGzEventSource(path, rep)

	tickCh, errCh, err := src.StreamTicks(context.Background())
	if err != nil {
		t.Fatalf("unexpected StreamTicks error: %v", err)
	}

	for b := range tickCh {
		_ = b
	}
	if err := <-errCh; err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	warnCount := 0
	for _, it := range rep.Items {
		if it.Level == "WARN" {
			warnCount++
		}
	}
	if warnCount < 3 {
		t.Errorf("expected at least 3 geometry warnings (window rect, ocr window rect, ocr block bounding box), got %d: %+v", warnCount, rep.Items)
	}
}
