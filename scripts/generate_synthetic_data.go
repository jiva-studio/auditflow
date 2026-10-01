package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func writeTarFile(tw *tar.Writer, name string, data []byte) error {
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

func generateSyntheticArchive(targetPath string) error {
	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	startTime := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	endTime := startTime.Add(15 * time.Second)

	metaJSON := fmt.Sprintf(`{
		"schema_version": "1.0.0",
		"session_id": "sess-synthetic-scenarios",
		"employee_id": "emp-synthetic",
		"started_at": "%s",
		"ended_at": "%s",
		"machine": {
			"hostname": "TEST-DESKTOP",
			"os_version": "Windows 11",
			"displays": [
				{"id": 0, "bounds": [0, 0, 1920, 1080], "scale": 1.0, "primary": true},
				{"id": 1, "bounds": [1920, 0, 2560, 1440], "scale": 1.0, "primary": false}
			]
		}
	}`, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))

	if err := writeTarFile(tw, "session/metadata.json", []byte(metaJSON)); err != nil {
		return err
	}

	ts1_ocr := startTime.Add(1*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts1_win := startTime.Add(1*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts1_click := startTime.Add(1*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr1 := fmt.Sprintf(`{"ts":"%s","filename":"s1.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"FW: Q3 Marketing Plan","bounding_box":[100,200,300,40],"confidence":1.0}]}`+"\n", ts1_ocr)
	win1 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - Outlook","process_name":"OUTLOOK.EXE","window_rect":[0,0,1920,1080]}`+"\n", ts1_win)
	mouse1 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":150,"mouse_y":220,"click_count":"single"}`+"\n", ts1_click)

	ts2_ocr := startTime.Add(2*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts2_win := startTime.Add(2*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts2_click := startTime.Add(2*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr2 := fmt.Sprintf(`{"ts":"%s","filename":"s2.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"FW: Urgent Security Notice","bounding_box":[100,300,300,40],"confidence":1.0}]}`+"\n", ts2_ocr)
	win2 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - olk.exe","process_name":"olk.exe","window_rect":[0,0,1920,1080]}`+"\n", ts2_win)
	mouse2 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":150,"mouse_y":320,"click_count":"single"}`+"\n", ts2_click)

	ts3_ocr := startTime.Add(3*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts3_win := startTime.Add(3*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts3_click := startTime.Add(3*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr3 := fmt.Sprintf(`{"ts":"%s","filename":"s3.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"FW: Notepad Notes","bounding_box":[100,400,300,40],"confidence":1.0}]}`+"\n", ts3_ocr)
	win3 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"notes.txt - Notepad","process_name":"notepad.exe","window_rect":[0,0,1920,1080]}`+"\n", ts3_win)
	mouse3 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":150,"mouse_y":420,"click_count":"single"}`+"\n", ts3_click)

	ts4_ocr := startTime.Add(4*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts4_win := startTime.Add(4*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts4_click := startTime.Add(4*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr4 := fmt.Sprintf(`{"ts":"%s","filename":"s4.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"Are you sure you want to delete this case?","bounding_box":[50,50,500,40],"confidence":1.0},{"text":"Delete","bounding_box":[200,500,80,30],"confidence":1.0}]}`+"\n", ts4_ocr)
	win4 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"0054321 | Case | Salesforce - Google Chrome","process_name":"chrome.exe","window_rect":[0,0,1920,1080]}`+"\n", ts4_win)
	mouse4 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":220,"mouse_y":510,"click_count":"single"}`+"\n", ts4_click)

	ts5_ocr := startTime.Add(5*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts5_click := startTime.Add(5*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ocr5 := fmt.Sprintf(`{"ts":"%s","filename":"s5.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"Are you sure you want to delete this case?","bounding_box":[50,50,500,40],"confidence":1.0},{"text":"Cancel","bounding_box":[300,500,80,30],"confidence":1.0}]}`+"\n", ts5_ocr)
	mouse5 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":320,"mouse_y":510,"click_count":"single"}`+"\n", ts5_click)

	ts6_ocr := startTime.Add(6*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts6_win := startTime.Add(6*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts6_click := startTime.Add(6*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr6 := fmt.Sprintf(`{"ts":"%s","filename":"s6.jpg","display_id":0,"resolution":[1920,1080],"ocr_text_blocks":[{"text":"Done","bounding_box":[100,100,80,30],"confidence":1.0}]}`+"\n", ts6_ocr)
	win6 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"DEV-909 - Jira - Mozilla Firefox","process_name":"firefox.exe","window_rect":[0,0,1920,1080]}`+"\n", ts6_win)
	mouse6 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":120,"mouse_y":110,"click_count":"single"}`+"\n", ts6_click)

	ts7_win := startTime.Add(7*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts7_clip := startTime.Add(7*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	win7 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Compose Message - Outlook","process_name":"OUTLOOK.EXE","window_rect":[0,0,1920,1080]}`+"\n", ts7_win)
	clip7 := fmt.Sprintf(`{"ts":"%s","event":"clipboard_change","clipboard_content_text":"INV-55443","clipboard_content_length":9}`+"\n", ts7_clip)

	ts8_win_excel := startTime.Add(8*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts8_clip_excel := startTime.Add(8*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts8_win_outlook := startTime.Add(8*time.Second + 500*time.Millisecond).Format(time.RFC3339Nano)
	win8_excel := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Invoices.xlsx - Excel","process_name":"EXCEL.EXE","window_rect":[0,0,1920,1080]}`+"\n", ts8_win_excel)
	clip8 := fmt.Sprintf(`{"ts":"%s","event":"clipboard_change","clipboard_content_text":"INV-88990","clipboard_content_length":9}`+"\n", ts8_clip_excel)
	win8_outlook := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - Outlook","process_name":"olk.exe","window_rect":[0,0,1920,1080]}`+"\n", ts8_win_outlook)

	ts9_win := startTime.Add(9*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts9_ocr := startTime.Add(9*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts9_click := startTime.Add(9*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	win9 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - Secondary Screen","process_name":"OUTLOOK.EXE","window_rect":[1920,0,2560,1440]}`+"\n", ts9_win)
	ocr9 := fmt.Sprintf(`{"ts":"%s","filename":"s9.jpg","display_id":1,"resolution":[2560,1440],"window_rect":[1920,0,2560,1440],"ocr_text_blocks":[{"text":"FW: Multi-Display Project Plan","bounding_box":[200,300,400,40],"confidence":1.0}]}`+"\n", ts9_ocr)
	mouse9 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":2170,"mouse_y":320,"click_count":"single"}`+"\n", ts9_click)

	ocrContent := ocr1 + ocr2 + ocr3 + ocr4 + ocr5 + ocr6 + ocr9
	winContent := win1 + win2 + win3 + win4 + win6 + win7 + win8_excel + win8_outlook + win9
	mouseContent := mouse1 + mouse2 + mouse3 + mouse4 + mouse5 + mouse6 + mouse9
	clipContent := clip7 + clip8

	if err := writeTarFile(tw, "session/ocr.jsonl", []byte(ocrContent)); err != nil {
		return err
	}
	if err := writeTarFile(tw, "session/windows.jsonl", []byte(winContent)); err != nil {
		return err
	}
	if err := writeTarFile(tw, "session/mouse.jsonl", []byte(mouseContent)); err != nil {
		return err
	}
	if err := writeTarFile(tw, "session/clipboard.jsonl", []byte(clipContent)); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gzw.Close()
}

func generateEdgeArchive(targetPath string) error {
	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	startTime := time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC)
	endTime := startTime.Add(20 * time.Second)

	metaJSON := fmt.Sprintf(`{
		"schema_version": "1.0.0",
		"session_id": "sess-edge-scenarios",
		"employee_id": "emp-edge",
		"started_at": "%s",
		"ended_at": "%s",
		"machine": {
			"hostname": "EDGE-DESKTOP",
			"os_version": "Windows 11 Enterprise",
			"displays": [
				{"id": 0, "bounds": [0, 0, 3840, 2160], "scale": 2.0, "primary": true}
			]
		}
	}`, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))

	if err := writeTarFile(tw, "session/metadata.json", []byte(metaJSON)); err != nil {
		return err
	}

	ts1_ocr := startTime.Add(1*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts1_win := startTime.Add(1*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts1_mouse := startTime.Add(1*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr1 := fmt.Sprintf(`{"ts":"%s","filename":"frame_1.jpg","display_id":0,"resolution":[3840,2160],"display_scale_factor":2.0,"ocr_text_blocks":[{"text":"FW: Right Click Ignored","bounding_box":[200,200,400,50],"confidence":1.0}]}`+"\n", ts1_ocr)
	win1 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - Outlook","process_name":"OUTLOOK.EXE","window_rect":[0,0,3840,2160]}`+"\n", ts1_win)
	mouse1 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"right","mouse_x":250,"mouse_y":220,"click_count":"single"}`+"\n", ts1_mouse)

	ts2_mouse := startTime.Add(2*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	mouse2 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":250,"mouse_y":220,"click_count":"single"}`+"\n", ts2_mouse)

	ts3_ocr := startTime.Add(3*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts3_mouse := startTime.Add(3*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr3 := fmt.Sprintf(`{"ts":"%s","filename":"frame_2.jpg","display_id":0,"resolution":[3840,2160],"deduplicated_from":"frame_1.jpg","ocr_text_blocks":[]}`+"\n", ts3_ocr)
	mouse3 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":250,"mouse_y":220,"click_count":"double"}`+"\n", ts3_mouse)

	ts4_ocr := startTime.Add(4*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts4_win := startTime.Add(4*time.Second + 150*time.Millisecond).Format(time.RFC3339Nano)
	ts4_click1 := startTime.Add(4*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts4_click2 := startTime.Add(4*time.Second + 500*time.Millisecond).Format(time.RFC3339Nano)
	ocr4 := fmt.Sprintf(`{"ts":"%s","filename":"frame_4.jpg","display_id":0,"resolution":[3840,2160],"ocr_text_blocks":[{"text":"Done","bounding_box":[100,100,100,40],"confidence":1.0},{"text":"Done","bounding_box":[100,300,100,40],"confidence":1.0}]}`+"\n", ts4_ocr)
	win4 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"BACKLOG-100 - Jira - Chrome","process_name":"chrome.exe","window_rect":[0,0,3840,2160]}`+"\n", ts4_win)
	mouse4_1 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":120,"mouse_y":110,"click_count":"single"}`+"\n", ts4_click1)
	mouse4_2 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":120,"mouse_y":310,"click_count":"single"}`+"\n", ts4_click2)

	ts5_ocr := startTime.Add(5*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts5_win := startTime.Add(5*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts5_click := startTime.Add(5*time.Second + 300*time.Millisecond).Format(time.RFC3339Nano)
	ocr5 := fmt.Sprintf(`{"ts":"%s","filename":"frame_5.jpg","display_id":0,"resolution":[3840,2160],"ocr_text_blocks":[{"text":"done","bounding_box":[500,500,80,30],"confidence":1.0}]}`+"\n", ts5_ocr)
	win5 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"feature-99 - jira - edge","process_name":"msedge.exe","window_rect":[0,0,3840,2160]}`+"\n", ts5_win)
	mouse5 := fmt.Sprintf(`{"ts":"%s","event":"click","button":"left","mouse_x":520,"mouse_y":510,"click_count":"single"}`+"\n", ts5_click)

	ts6_win := startTime.Add(6*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	ts6_clip_neg := startTime.Add(6*time.Second + 200*time.Millisecond).Format(time.RFC3339Nano)
	ts6_clip_pos := startTime.Add(6*time.Second + 500*time.Millisecond).Format(time.RFC3339Nano)
	win6 := fmt.Sprintf(`{"ts":"%s","event":"focus_change","window_title":"Inbox - Outlook","process_name":"OUTLOOK.EXE","window_rect":[0,0,3840,2160]}`+"\n", ts6_win)
	clip6_neg := fmt.Sprintf(`{"ts":"%s","event":"clipboard_change","clipboard_content_text":"INVOICE-99999","clipboard_content_length":13}`+"\n", ts6_clip_neg)
	clip6_pos := fmt.Sprintf(`{"ts":"%s","event":"clipboard_change","clipboard_content_text":"INV-001122","clipboard_content_length":10}`+"\n", ts6_clip_pos)

	ts7_key := startTime.Add(7*time.Second + 100*time.Millisecond).Format(time.RFC3339Nano)
	key7 := fmt.Sprintf(`{"ts":"%s","event":"key_press","key":"KEY_ENTER"}`+"\n", ts7_key)

	ocrContent := ocr1 + ocr3 + ocr4 + ocr5
	winContent := win1 + win4 + win5 + win6
	mouseContent := mouse1 + mouse2 + mouse3 + mouse4_1 + mouse4_2 + mouse5
	clipContent := clip6_neg + clip6_pos
	keyContent := key7

	if err := writeTarFile(tw, "session/ocr.jsonl", []byte(ocrContent)); err != nil {
		return err
	}
	if err := writeTarFile(tw, "session/windows.jsonl", []byte(winContent)); err != nil {
		return err
	}
	if err := writeTarFile(tw, "session/mouse.jsonl", []byte(mouseContent)); err != nil {
		return err
	}
	if err := writeTarFile(tw, "session/clipboard.jsonl", []byte(clipContent)); err != nil {
		return err
	}
	if err := writeTarFile(tw, "session/keyboard.jsonl", []byte(keyContent)); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gzw.Close()
}

func main() {
	rootDir := "data"
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	synthPath := filepath.Join(rootDir, "emp-synthetic.tar.gz")
	if err := generateSyntheticArchive(synthPath); err != nil {
		log.Fatalf("failed to generate synthetic archive: %v", err)
	}
	log.Printf("Generated %s successfully", synthPath)

	edgePath := filepath.Join(rootDir, "emp-edge.tar.gz")
	if err := generateEdgeArchive(edgePath); err != nil {
		log.Fatalf("failed to generate edge archive: %v", err)
	}
	log.Printf("Generated %s successfully", edgePath)
}
