package testutil

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// GenerateSyntheticRecording creates a multi-stream .tar.gz recording on disk
// using streaming I/O with O(1) memory overhead.
func GenerateSyntheticRecording(t *testing.T, targetPath string, numSeconds, eventsPerSec int) (int64, error) {
	t.Helper()

	f, err := os.Create(targetPath)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = f.Close()
	}()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	startTime := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	endTime := startTime.Add(time.Duration(numSeconds) * time.Second)

	// 1. Write session/metadata.json
	metaJSON := fmt.Sprintf(`{
		"schema_version": "1.0.0",
		"session_id": "sess-e2e-stress",
		"employee_id": "emp-e2e-stress",
		"started_at": "%s",
		"ended_at": "%s",
		"machine": {
			"hostname": "STRESS-HOST",
			"os_version": "Linux 6.6",
			"displays": [
				{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}
			]
		}
	}`, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))

	if err := writeTarEntry(tw, "session/metadata.json", []byte(metaJSON)); err != nil {
		return 0, err
	}

	// 2. Stream event files directly through buffered temp files
	tmpDir := t.TempDir()
	streams := []string{
		"session/windows.jsonl",
		"session/mouse.jsonl",
		"session/clipboard.jsonl",
		"session/keyboard.jsonl",
		"session/ocr.jsonl",
	}

	for _, streamName := range streams {
		baseName := filepath.Base(streamName)
		tmpStreamPath := filepath.Join(tmpDir, baseName)
		if err := writeStreamDataToFile(tmpStreamPath, baseName, startTime, numSeconds, eventsPerSec/len(streams)); err != nil {
			return 0, err
		}

		info, err := os.Stat(tmpStreamPath)
		if err != nil {
			return 0, err
		}

		hdr := &tar.Header{
			Name: streamName,
			Mode: 0644,
			Size: info.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return 0, err
		}

		streamFile, err := os.Open(tmpStreamPath)
		if err != nil {
			return 0, err
		}
		if _, err := streamFile.WriteTo(tw); err != nil {
			_ = streamFile.Close()
			return 0, err
		}
		_ = streamFile.Close()
		_ = os.Remove(tmpStreamPath)
	}

	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gzw.Close(); err != nil {
		return 0, err
	}

	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
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

func writeStreamDataToFile(path, streamType string, startTime time.Time, numSeconds, perSec int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	bw := bufio.NewWriterSize(f, 4*1024*1024)

	sampleWords := []string{
		"CONFIDENTIAL", "FINANCIAL", "STATEMENT", "TRANSACTION", "INVOICE",
		"LEDGER", "AUTHORIZED", "SIGNATURE", "ENTERPRISE", "ACCOUNTING",
		"COMPLIANCE", "VERIFICATION", "AUDIT", "PAYROLL", "DISBURSEMENT",
		"RECONCILIATION", "CONTRACT", "AGREEMENT", "SCHEDULE", "BALANCE",
	}

	for s := 0; s < numSeconds; s++ {
		secBase := startTime.Add(time.Duration(s) * time.Second)
		for e := 0; e < perSec; e++ {
			ts := secBase.Add(time.Duration(e*2) * time.Millisecond).Format(time.RFC3339Nano)
			var line string
			switch streamType {
			case "windows.jsonl":
				w1 := sampleWords[(s+e)%len(sampleWords)]
				w2 := sampleWords[(s*3+e)%len(sampleWords)]
				line = fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"%s - %s Workstation Enterprise Suite [%x-%x]","process_name":"app_%d.exe","window_rect":[0,0,1920,1080]}`+"\n", ts, w1, w2, s*1000+e, e*7, e%50)
			case "mouse.jsonl":
				line = fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":%d,"mouse_y":%d,"click_count":"single"}`+"\n", ts, (s*7+e)%1920, (s*11+e)%1080)
			case "clipboard.jsonl":
				w := sampleWords[(s+e*2)%len(sampleWords)]
				line = fmt.Sprintf(`{"ts":"%s","event":"clipboard_change","clipboard_content_text":"%s-PAYLOAD-TRANSACTION-DATA-BLOCK-%x-%x-%d-SECURE-HASH-RECORD","clipboard_content_length":128}`+"\n", ts, w, s*100000+e, s*777+e, e)
			case "keyboard.jsonl":
				line = fmt.Sprintf(`{"ts":"%s","event":"key_press","key":"KEY_%c"}`+"\n", ts, 'A'+(e%26))
			case "ocr.jsonl":
				w1 := sampleWords[(s+e)%len(sampleWords)]
				w2 := sampleWords[(s+e+1)%len(sampleWords)]
				w3 := sampleWords[(s+e+2)%len(sampleWords)]
				w4 := sampleWords[(s+e+3)%len(sampleWords)]
				w5 := sampleWords[(s+e+4)%len(sampleWords)]
				w6 := sampleWords[(s+e+5)%len(sampleWords)]
				line = fmt.Sprintf(`{"ts":"%s","filename":"doc_%x_%x.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"%s %s INVOICE #%x-%x TOTAL DUE: $%.2f PAYMENT TERMS: NET30 VENDOR ACME ENTERPRISES","bounding_box":[10,20,500,50],"confidence":0.99},{"text":"%s %s APPROVED FOR AUDIT RECORD ID %x VERIFICATION TIMESTAMP %s OFFICER #%d","bounding_box":[10,80,600,40],"confidence":0.98},{"text":"%s %s COMPLIANCE AND TAX LEDGER IDENTIFIER %x-%x ROUTING TRANSIT CODE %x","bounding_box":[10,130,550,45],"confidence":0.97}]}`+"\n",
					ts, s, e, w1, w2, s, e, float64(s*100+e)*1.25, w3, w4, (s*999999 + e), ts, e%100, w5, w6, s*333+e, e*11, s*98765+e)
			}
			if _, err := bw.WriteString(line); err != nil {
				return err
			}
		}
	}
	return bw.Flush()
}

// GenerateTemplatePlaceholderRecording creates a synthetic recording specifically designed to trigger
// rules with placeholder templates where some variables are resolved and others are omitted/unresolved.
func GenerateTemplatePlaceholderRecording(t *testing.T, targetPath string, employeeID string) (int64, error) {
	t.Helper()

	f, err := os.Create(targetPath)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = f.Close()
	}()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	startTime := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Second)

	// 1. Write session/metadata.json
	metaJSON := fmt.Sprintf(`{
		"schema_version": "1.0.0",
		"session_id": "sess-placeholder-test",
		"employee_id": "%s",
		"started_at": "%s",
		"ended_at": "%s",
		"machine": {
			"hostname": "PLACEHOLDER-HOST",
			"os_version": "Linux 6.6",
			"displays": [
				{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}
			]
		}
	}`, employeeID, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))

	if err := writeTarEntry(tw, "session/metadata.json", []byte(metaJSON)); err != nil {
		return 0, err
	}

	// 2. Write event streams
	ts1 := startTime.Add(100 * time.Millisecond).Format(time.RFC3339Nano)
	windowsJSONL := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - Outlook","process_name":"OUTLOOK.EXE","window_rect":[0,0,1920,1080]}`+"\n", ts1)
	if err := writeTarEntry(tw, "session/windows.jsonl", []byte(windowsJSONL)); err != nil {
		return 0, err
	}

	ts2 := startTime.Add(200 * time.Millisecond).Format(time.RFC3339Nano)
	ocrJSONL := fmt.Sprintf(`{"ts":"%s","filename":"screen_0.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"FW: Project Alpha Update","bounding_box":[100,100,300,50],"confidence":0.99}]}`+"\n", ts2)
	if err := writeTarEntry(tw, "session/ocr.jsonl", []byte(ocrJSONL)); err != nil {
		return 0, err
	}

	ts3 := startTime.Add(300 * time.Millisecond).Format(time.RFC3339Nano)
	mouseJSONL := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":150,"mouse_y":120,"click_count":"single"}`+"\n", ts3)
	if err := writeTarEntry(tw, "session/mouse.jsonl", []byte(mouseJSONL)); err != nil {
		return 0, err
	}

	if err := writeTarEntry(tw, "session/clipboard.jsonl", []byte{}); err != nil {
		return 0, err
	}
	if err := writeTarEntry(tw, "session/keyboard.jsonl", []byte{}); err != nil {
		return 0, err
	}

	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gzw.Close(); err != nil {
		return 0, err
	}

	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}
