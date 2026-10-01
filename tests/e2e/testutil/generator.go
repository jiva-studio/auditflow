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

// GenerateLeftHandedMouseRecording creates a synthetic recording containing left-handed primary clicks,
// move gestures, and non-primary mouse clicks for E2E testing.
func GenerateLeftHandedMouseRecording(t *testing.T, targetPath string) error {
	t.Helper()

	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	startTime := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	endTime := startTime.Add(5 * time.Second)

	metaJSON := fmt.Sprintf(`{
		"schema_version": "1.0.0",
		"session_id": "sess-lefthanded-mouse",
		"employee_id": "emp-lefthanded",
		"started_at": "%s",
		"ended_at": "%s",
		"machine": {
			"hostname": "LEFT-WORKSTATION",
			"os_version": "Linux 6.6",
			"displays": [
				{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}
			]
		}
	}`, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))

	if err := writeTarEntry(tw, "session/metadata.json", []byte(metaJSON)); err != nil {
		return err
	}

	// 1. Second 1: Outlook - Forwarded Email with Left-Handed "primary" button click and move/middle gestures
	ts1_ocr := startTime.Add(1*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts1_win := startTime.Add(1*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts1_move := startTime.Add(1*time.Second + 250*time.Millisecond).Format(time.RFC3339Nano)
	ts1_middle := startTime.Add(1*time.Second + 270*time.Millisecond).Format(time.RFC3339Nano)
	ts1_click := startTime.Add(1*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)

	ocr1 := fmt.Sprintf(`{"ts":"%s","filename":"s1.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"FW: Left Handed Budget Update","bounding_box":[100,200,350,40],"confidence":1.0}]}`+"\n", ts1_ocr)
	win1 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - Outlook","process_name":"OUTLOOK.EXE","window_rect":[0,0,1920,1080]}`+"\n", ts1_win)
	mouse1_move := fmt.Sprintf(`{"ts":"%s","event":"move","button":"primary","mouse_x":150,"mouse_y":220,"click_count":"none"}`+"\n", ts1_move)
	mouse1_middle := fmt.Sprintf(`{"ts":"%s","event":"click","button":"middle","mouse_x":150,"mouse_y":220,"click_count":"single"}`+"\n", ts1_middle)
	mouse1_click := fmt.Sprintf(`{"ts":"%s","event":"click","button":"primary","mouse_x":150,"mouse_y":220,"click_count":"single"}`+"\n", ts1_click)

	// 2. Second 2: Jira Done with "main" button click and drag gestures
	ts2_ocr := startTime.Add(2*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts2_win := startTime.Add(2*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts2_drag := startTime.Add(2*time.Second + 250*time.Millisecond).Format(time.RFC3339Nano)
	ts2_click := startTime.Add(2*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)

	ocr2 := fmt.Sprintf(`{"ts":"%s","filename":"s2.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"Done","bounding_box":[100,100,80,30],"confidence":1.0}]}`+"\n", ts2_ocr)
	win2 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"DEV-42 - Jira - Google Chrome","process_name":"chrome.exe","window_rect":[0,0,1920,1080]}`+"\n", ts2_win)
	mouse2_drag := fmt.Sprintf(`{"ts":"%s","event":"drag","button":"main","mouse_x":120,"mouse_y":110,"click_count":"none"}`+"\n", ts2_drag)
	mouse2_click := fmt.Sprintf(`{"ts":"%s","event":"click","button":"main","mouse_x":120,"mouse_y":110,"click_count":"single"}`+"\n", ts2_click)

	// 3. Second 3: Urgent email with "mousedown" action and scroll gestures
	ts3_ocr := startTime.Add(3*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts3_win := startTime.Add(3*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts3_scroll := startTime.Add(3*time.Second + 250*time.Millisecond).Format(time.RFC3339Nano)
	ts3_click := startTime.Add(3*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)

	ocr3 := fmt.Sprintf(`{"ts":"%s","filename":"s3.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"Urgent Security Patch Review","bounding_box":[200,300,350,40],"confidence":1.0}]}`+"\n", ts3_ocr)
	win3 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - olk.exe","process_name":"olk.exe","window_rect":[0,0,1920,1080]}`+"\n", ts3_win)
	mouse3_scroll := fmt.Sprintf(`{"ts":"%s","event":"scroll","button":"primary","mouse_x":250,"mouse_y":320,"click_count":"none"}`+"\n", ts3_scroll)
	mouse3_click := fmt.Sprintf(`{"ts":"%s","event":"mousedown","button":"primary","mouse_x":250,"mouse_y":320,"click_count":"single"}`+"\n", ts3_click)

	ocrContent := ocr1 + ocr2 + ocr3
	winContent := win1 + win2 + win3
	mouseContent := mouse1_move + mouse1_middle + mouse1_click + mouse2_drag + mouse2_click + mouse3_scroll + mouse3_click

	if err := writeTarEntry(tw, "session/ocr.jsonl", []byte(ocrContent)); err != nil {
		return err
	}
	if err := writeTarEntry(tw, "session/windows.jsonl", []byte(winContent)); err != nil {
		return err
	}
	if err := writeTarEntry(tw, "session/mouse.jsonl", []byte(mouseContent)); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gzw.Close()
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

// GenerateClickProcessContextRecording creates a synthetic recording archive that tests
// click evaluation using process and window title context directly from mouse events.
func GenerateClickProcessContextRecording(t *testing.T, targetPath, employeeID string) (int64, error) {
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

	metaJSON := fmt.Sprintf(`{
		"schema_version": "1.0.0",
		"session_id": "sess-e2e-click-context",
		"employee_id": "%s",
		"started_at": "%s",
		"ended_at": "%s",
		"machine": {
			"hostname": "CLICK-CONTEXT-HOST",
			"os_version": "Linux 6.6",
			"displays": [
				{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true}
			]
		}
	}`, employeeID, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))

	if err := writeTarEntry(tw, "session/metadata.json", []byte(metaJSON)); err != nil {
		return 0, err
	}

	windowsData := fmt.Sprintf(
		`{"ts":"%s","event":"focus_change","window_title":"Background Explorer","process_name":"explorer.exe","window_rect":[0,0,1920,1080]}`+"\n",
		startTime.Add(100*time.Millisecond).Format(time.RFC3339Nano),
	)
	if err := writeTarEntry(tw, "session/windows.jsonl", []byte(windowsData)); err != nil {
		return 0, err
	}

	ocrData := fmt.Sprintf(
		`{"ts":"%s","filename":"frame1.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"FW: Q3 Audit Report","bounding_box":[100,100,250,30],"confidence":0.99},{"text":"Done","bounding_box":[400,200,100,30],"confidence":0.99}]}`+"\n",
		startTime.Add(200*time.Millisecond).Format(time.RFC3339Nano),
	)
	if err := writeTarEntry(tw, "session/ocr.jsonl", []byte(ocrData)); err != nil {
		return 0, err
	}

	mouseData := fmt.Sprintf(
		`{"ts":"%s","event":"click","button":"left","mouse_x":150,"mouse_y":110,"click_count":"1","process_name":"OUTLOOK.EXE","window_title":"Inbox - Outlook"}`+"\n"+
			`{"ts":"%s","event":"click","button":"left","mouse_x":420,"mouse_y":210,"click_count":"1","process_name":"chrome.exe","window_title":"PROJ-101 - Jira - Dashboard"}`+"\n",
		startTime.Add(500*time.Millisecond).Format(time.RFC3339Nano),
		startTime.Add(1500*time.Millisecond).Format(time.RFC3339Nano),
	)
	if err := writeTarEntry(tw, "session/mouse.jsonl", []byte(mouseData)); err != nil {
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
