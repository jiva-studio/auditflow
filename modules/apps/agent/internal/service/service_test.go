package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"assessment/modules/libs/domain/audit"
	"assessment/modules/libs/domain/desktop"
	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/geometry"
	"assessment/modules/libs/domain/rules"
)

type mockAuditClient struct {
	mu      sync.Mutex
	popups  []audit.Popup
	sendErr error
}

func (m *mockAuditClient) SendPopup(_ context.Context, popup audit.Popup) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.popups = append(m.popups, popup)
	return nil
}

func (m *mockAuditClient) Popups() []audit.Popup {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]audit.Popup, len(m.popups))
	copy(res, m.popups)
	return res
}

func sampleTestRules(t *testing.T) []rules.Rule {
	t.Helper()
	clickPat, _ := rules.NewPatternList("FW:*")
	procPat, _ := rules.NewPatternList("OUTLOOK.EXE", "olk.exe")
	tpl1, _ := rules.NewPopupTemplate("Forwarded email", "Opened: {click}")

	r1, err := rules.NewRule("forwarded-email", rules.WhenConditions{
		Click:   &clickPat,
		Process: &procPat,
	}, tpl1)
	if err != nil {
		t.Fatalf("failed to create rule 1: %v", err)
	}

	clipPat, _ := rules.NewPatternList("INV-*")
	procPat2, _ := rules.NewPatternList("OUTLOOK.EXE")
	tpl2, _ := rules.NewPopupTemplate("Invoice in clipboard", "Copied: {clipboard}")

	r2, err := rules.NewRule("invoice-clipboard", rules.WhenConditions{
		Clipboard: &clipPat,
		Process:   &procPat2,
	}, tpl2)
	if err != nil {
		t.Fatalf("failed to create rule 2: %v", err)
	}

	return []rules.Rule{r1, r2}
}

func TestNewService(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)

	t.Run("empty employee ID", func(t *testing.T) {
		_, err := NewService("", nil, rls, client)
		if !errors.Is(err, ErrEmptyEmployeeID) {
			t.Errorf("expected ErrEmptyEmployeeID, got %v", err)
		}
	})

	t.Run("nil audit client", func(t *testing.T) {
		_, err := NewService("emp-1", nil, rls, nil)
		if !errors.Is(err, ErrNilAuditClient) {
			t.Errorf("expected ErrNilAuditClient, got %v", err)
		}
	})

	t.Run("valid with default state", func(t *testing.T) {
		svc, err := NewService("emp-1", nil, rls, client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc == nil || svc.state == nil {
			t.Fatal("expected non-nil service and state")
		}
	})
}

func TestService_ProcessTick_ClickMatch(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, err := NewService("emp-1", desktop.NewState(), rls, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	// 1. OCR Event with text "FW: Important Notice"
	_ = batch.Add(events.OCREvent{
		Timestamp:  now,
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "FW: Important Notice",
				Box:  geometry.Rectangle{X: 100, Y: 100, Width: 200, Height: 30},
			},
		},
	})

	// 2. Window Event focusing Outlook
	_ = batch.Add(events.WindowEvent{
		Timestamp:   now.Add(10 * time.Millisecond),
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	// 3. Mouse Click on the OCR text
	_ = batch.Add(events.MouseEvent{
		Timestamp:   now.Add(20 * time.Millisecond),
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 150, Y: 110},
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	ctx := context.Background()
	if err := svc.ProcessTick(ctx, batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 1 {
		t.Fatalf("expected 1 popup, got %d", len(popups))
	}
	p := popups[0]
	if p.Employee != "emp-1" || p.Rule != "forwarded-email" || p.Title != "Forwarded email" || p.Body != "Opened: FW: Important Notice" {
		t.Errorf("unexpected popup details: %+v", p)
	}
}

func TestService_ProcessTick_ClipboardMatch(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, err := NewService("emp-2", desktop.NewState(), rls, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	// Window Event focusing Outlook
	_ = batch.Add(events.WindowEvent{
		Timestamp:   now,
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "New Mail",
	})

	// Clipboard Event copying INV-8877
	_ = batch.Add(events.ClipboardEvent{
		Timestamp: now.Add(10 * time.Millisecond),
		Action:    "copy",
		Text:      "INV-8877",
	})

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 1 {
		t.Fatalf("expected 1 popup, got %d", len(popups))
	}
	if popups[0].Rule != "invoice-clipboard" || popups[0].Body != "Copied: INV-8877" {
		t.Errorf("unexpected popup: %+v", popups[0])
	}
}

func TestService_ProcessTick_IgnoredEventsAndFailures(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, err := NewService("emp-1", desktop.NewState(), rls, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	// Generic Activity Event
	_ = batch.Add(events.GenericActivityEvent{
		Timestamp: now,
		Type:      events.EventTypeKeystroke,
	})

	// Mouse Move without click
	_ = batch.Add(events.MouseEvent{
		Timestamp: now.Add(10 * time.Millisecond),
		Action:    "move",
		Button:    "none",
		Position:  geometry.Point{X: 10, Y: 10},
	})

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	if len(client.Popups()) != 0 {
		t.Fatalf("expected 0 popups, got %d", len(client.Popups()))
	}

	// Test client dispatch failure
	client.sendErr = errors.New("network down")
	batch2, _ := events.NewTickBatch(1, now.Add(time.Second), now.Add(2*time.Second))
	_ = batch2.Add(events.WindowEvent{Timestamp: now.Add(time.Second), ProcessName: "OUTLOOK.EXE"})
	_ = batch2.Add(events.ClipboardEvent{Timestamp: now.Add(time.Second + 10*time.Millisecond), Text: "INV-999"})

	err = svc.ProcessTick(context.Background(), batch2)
	if !errors.Is(err, ErrDispatchPopupFail) {
		t.Errorf("expected ErrDispatchPopupFail, got %v", err)
	}
}

func TestService_Concurrency(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, _ := NewService("emp-1", desktop.NewState(), rls, client)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			now := time.Now().UTC().Add(time.Duration(idx) * time.Minute)
			batch, _ := events.NewTickBatch(idx, now, now.Add(time.Second))
			_ = batch.Add(events.WindowEvent{
				Timestamp:   now,
				ProcessName: "notepad.exe",
			})
			_ = svc.ProcessTick(context.Background(), batch)
		}(i)
	}
	wg.Wait()
}

func TestService_CheckReadiness(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockAuditClient{}
		rls := sampleTestRules(t)
		svc, err := NewService("emp-1", nil, rls, client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := svc.CheckReadiness(context.Background()); err != nil {
			t.Fatalf("expected readiness check to succeed, got: %v", err)
		}
	})

	t.Run("no rules loaded", func(t *testing.T) {
		client := &mockAuditClient{}
		svc, err := NewService("emp-1", nil, nil, client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := svc.CheckReadiness(context.Background()); !errors.Is(err, ErrNoRulesLoaded) {
			t.Fatalf("expected ErrNoRulesLoaded, got: %v", err)
		}
	})
}

func TestIsClickEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    events.MouseEvent
		expected bool
	}{
		// Primary button clicks
		{name: "standard left click", event: events.MouseEvent{Action: "click", Button: "left"}, expected: true},
		{name: "primary button click", event: events.MouseEvent{Action: "click", Button: "primary"}, expected: true},
		{name: "main button click", event: events.MouseEvent{Action: "click", Button: "main"}, expected: true},
		{name: "mousedown action", event: events.MouseEvent{Action: "mousedown", Button: "primary"}, expected: true},
		{name: "mouse_click action", event: events.MouseEvent{Action: "mouse_click", Button: "primary"}, expected: true},
		{name: "left button without action", event: events.MouseEvent{Button: "left"}, expected: true},
		{name: "primary button without action", event: events.MouseEvent{Button: "primary"}, expected: true},
		{name: "main button without action", event: events.MouseEvent{Button: "main"}, expected: true},
		{name: "click action without button", event: events.MouseEvent{Action: "click"}, expected: true},
		{name: "mousedown action without button", event: events.MouseEvent{Action: "mousedown"}, expected: true},
		{name: "mouse_click action without button", event: events.MouseEvent{Action: "mouse_click"}, expected: true},
		{name: "click count single", event: events.MouseEvent{ClickCount: "single"}, expected: true},
		{name: "click count double", event: events.MouseEvent{ClickCount: "double"}, expected: true},
		{name: "click count 1", event: events.MouseEvent{ClickCount: "1"}, expected: true},
		{name: "case insensitive primary button", event: events.MouseEvent{Action: "CLICK", Button: "PRIMARY"}, expected: true},

		// Move & gesture rejections
		{name: "move action rejected", event: events.MouseEvent{Action: "move", Button: "primary"}, expected: false},
		{name: "mousemove action rejected", event: events.MouseEvent{Action: "mousemove", Button: "left"}, expected: false},
		{name: "drag action rejected", event: events.MouseEvent{Action: "drag", Button: "left"}, expected: false},
		{name: "mousedrag action rejected", event: events.MouseEvent{Action: "mousedrag", Button: "primary"}, expected: false},
		{name: "scroll action rejected", event: events.MouseEvent{Action: "scroll", Button: "primary"}, expected: false},
		{name: "move action with click count rejected", event: events.MouseEvent{Action: "move", ClickCount: "single"}, expected: false},
		{name: "drag action with primary button rejected", event: events.MouseEvent{Action: "drag", Button: "primary", ClickCount: "1"}, expected: false},

		// Middle click rejections
		{name: "middle click rejected", event: events.MouseEvent{Action: "click", Button: "middle"}, expected: false},
		{name: "middle mousedown rejected", event: events.MouseEvent{Action: "mousedown", Button: "middle"}, expected: false},
		{name: "middle mouse_click rejected", event: events.MouseEvent{Action: "mouse_click", Button: "middle"}, expected: false},
		{name: "middle button without action rejected", event: events.MouseEvent{Button: "middle"}, expected: false},

		// Right click rejections
		{name: "right click rejected", event: events.MouseEvent{Action: "click", Button: "right"}, expected: false},
		{name: "right mousedown rejected", event: events.MouseEvent{Action: "mousedown", Button: "right"}, expected: false},
		{name: "right mouse_click rejected", event: events.MouseEvent{Action: "mouse_click", Button: "right"}, expected: false},
		{name: "right button without action rejected", event: events.MouseEvent{Button: "right"}, expected: false},

		// Click count zero / none rejections
		{name: "empty event rejected", event: events.MouseEvent{}, expected: false},
		{name: "zero click count rejected", event: events.MouseEvent{ClickCount: "0"}, expected: false},
		{name: "none click count rejected", event: events.MouseEvent{ClickCount: "none"}, expected: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual := isClickEvent(tc.event)
			if actual != tc.expected {
				t.Errorf("isClickEvent(%+v) = %v, want %v", tc.event, actual, tc.expected)
			}
		})
	}
}

func TestService_ProcessTick_LeftHandedMouseClicks(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, err := NewService("emp-lefty", desktop.NewState(), rls, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	// OCR text
	_ = batch.Add(events.OCREvent{
		Timestamp:  now,
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "FW: Left-Handed Click Test",
				Box:  geometry.Rectangle{X: 100, Y: 100, Width: 200, Height: 30},
			},
		},
	})

	// Window focus Outlook
	_ = batch.Add(events.WindowEvent{
		Timestamp:   now.Add(10 * time.Millisecond),
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	// Move gesture (should be ignored)
	_ = batch.Add(events.MouseEvent{
		Timestamp:   now.Add(15 * time.Millisecond),
		Action:      "move",
		Button:      "primary",
		Position:    geometry.Point{X: 150, Y: 110},
		ProcessName: "OUTLOOK.EXE",
	})

	// Left-handed primary click
	_ = batch.Add(events.MouseEvent{
		Timestamp:   now.Add(20 * time.Millisecond),
		Action:      "click",
		Button:      "primary",
		Position:    geometry.Point{X: 150, Y: 110},
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	ctx := context.Background()
	if err := svc.ProcessTick(ctx, batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 1 {
		t.Fatalf("expected 1 popup, got %d", len(popups))
	}
	p := popups[0]
	if p.Employee != "emp-lefty" || p.Rule != "forwarded-email" || p.Title != "Forwarded email" || p.Body != "Opened: FW: Left-Handed Click Test" {
		t.Errorf("unexpected popup details: %+v", p)
	}
}

func runChaosBatch(workerID int, acts, btns, ccs []string) events.TickBatch {
	now := time.Now().UTC().Add(time.Duration(workerID) * time.Minute)
	batch, _ := events.NewTickBatch(workerID, now, now.Add(time.Second))
	_ = batch.Add(events.OCREvent{
		Timestamp:  now,
		DisplayID:  workerID % 3,
		Resolution: geometry.Size{Width: 3840, Height: 2160},
		Blocks: []display.OCRTextBlock{{
			Text: fmt.Sprintf("FW: Chaos %d \u200b 💣", workerID),
			Box:  geometry.Rectangle{X: -500 + workerID*10, Y: -500 + workerID*10, Width: 300, Height: 40},
		}},
	})
	_ = batch.Add(events.WindowEvent{
		Timestamp:   now.Add(5 * time.Millisecond),
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: fmt.Sprintf("Inbox - Outlook \u200b [%d] 🔥", workerID),
	})
	for j := 0; j < len(acts); j++ {
		_ = batch.Add(events.MouseEvent{
			Timestamp:   now.Add(time.Duration(10+j) * time.Millisecond),
			Action:      acts[(workerID+j)%len(acts)],
			Button:      btns[(workerID+j*2)%len(btns)],
			ClickCount:  ccs[(workerID+j*3)%len(ccs)],
			Position:    geometry.Point{X: -9999 + workerID*100 + j, Y: -9999 + workerID*50 + j},
			ProcessName: "OUTLOOK.EXE",
		})
	}
	return batch
}

func TestService_ChaosAndStress(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, err := NewService("emp-chaos", desktop.NewState(), rls, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	acts := []string{"click", "CLICK", "mousedown", "move", "drag", "scroll", "", "\u200b", "🔥"}
	btns := []string{"primary", "PRIMARY", "main", "left", "right", "middle", "", "\u200b", "🚀"}
	ccs := []string{"1", "single", "double", "0", "none", "", "triple"}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			b := runChaosBatch(wID, acts, btns, ccs)
			_ = svc.ProcessTick(context.Background(), b)
		}(i)
	}
	wg.Wait()
}

func setupOverrideTestRules() []rules.Rule {
	clickPat, _ := rules.NewPatternList("FW:*")
	procPat, _ := rules.NewPatternList("OUTLOOK.EXE")
	tpl1, _ := rules.NewPopupTemplate("Forwarded email", "Opened: {click}")
	r1, _ := rules.NewRule("forwarded-email", rules.WhenConditions{
		Click:   &clickPat,
		Process: &procPat,
	}, tpl1)

	savePat, _ := rules.NewPatternList("Save")
	winPat, _ := rules.NewPatternList("*Customer Record*")
	crmPat, _ := rules.NewPatternList("crm.exe")
	tpl2, _ := rules.NewPopupTemplate("Saved record", "Saved in window: {window_title}")
	r2, _ := rules.NewRule("save-crm", rules.WhenConditions{
		Click:       &savePat,
		Process:     &crmPat,
		WindowTitle: &winPat,
	}, tpl2)

	return []rules.Rule{r1, r2}
}

func TestService_ProcessTick_ClickProcessNameOverride(t *testing.T) {
	client := &mockAuditClient{}
	svc, err := NewService("emp-1", desktop.NewState(), setupOverrideTestRules(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	// Background window is Chrome
	_ = batch.Add(events.WindowEvent{
		Timestamp:   now,
		ProcessName: "chrome.exe",
		WindowTitle: "Google Chrome",
	})

	// OCR layer has "FW: Important Update"
	_ = batch.Add(events.OCREvent{
		Timestamp:  now.Add(5 * time.Millisecond),
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "FW: Important Update",
				Box:  geometry.Rectangle{X: 100, Y: 100, Width: 200, Height: 30},
			},
		},
	})

	// Mouse click event carrying Outlook process context
	_ = batch.Add(events.MouseEvent{
		Timestamp:   now.Add(10 * time.Millisecond),
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 150, Y: 110},
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 1 {
		t.Fatalf("expected 1 popup, got %d", len(popups))
	}
	if popups[0].Rule != "forwarded-email" || popups[0].Body != "Opened: FW: Important Update" {
		t.Errorf("unexpected popup: %+v", popups[0])
	}
}

func TestService_ProcessTick_ClickProcessNameMismatch(t *testing.T) {
	client := &mockAuditClient{}
	svc, err := NewService("emp-1", desktop.NewState(), setupOverrideTestRules(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	// Background window is Outlook (would match if used)
	_ = batch.Add(events.WindowEvent{
		Timestamp:   now,
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	_ = batch.Add(events.OCREvent{
		Timestamp:  now.Add(5 * time.Millisecond),
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "FW: Important Update",
				Box:  geometry.Rectangle{X: 100, Y: 100, Width: 200, Height: 30},
			},
		},
	})

	// Mouse event occurred in notepad.exe
	_ = batch.Add(events.MouseEvent{
		Timestamp:   now.Add(10 * time.Millisecond),
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 150, Y: 110},
		ProcessName: "notepad.exe",
		WindowTitle: "Untitled - Notepad",
	})

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 0 {
		t.Fatalf("expected 0 popups due to process mismatch, got %d", len(popups))
	}
}

func TestService_ProcessTick_ClickWindowTitleOverride(t *testing.T) {
	client := &mockAuditClient{}
	svc, err := NewService("emp-1", desktop.NewState(), setupOverrideTestRules(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	// Background window title does not match "*Customer Record*"
	_ = batch.Add(events.WindowEvent{
		Timestamp:   now,
		ProcessName: "crm.exe",
		WindowTitle: "Settings Dashboard",
	})

	_ = batch.Add(events.OCREvent{
		Timestamp:  now.Add(5 * time.Millisecond),
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "Save",
				Box:  geometry.Rectangle{X: 300, Y: 300, Width: 80, Height: 30},
			},
		},
	})

	// Mouse event has WindowTitle matching "*Customer Record*"
	_ = batch.Add(events.MouseEvent{
		Timestamp:   now.Add(10 * time.Millisecond),
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 320, Y: 310},
		ProcessName: "crm.exe",
		WindowTitle: "Edit - Customer Record #42",
	})

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 1 {
		t.Fatalf("expected 1 popup, got %d", len(popups))
	}
	if popups[0].Rule != "save-crm" || popups[0].Body != "Saved in window: Edit - Customer Record #42" {
		t.Errorf("unexpected popup: %+v", popups[0])
	}
}

func TestService_ProcessTick_ClickFallbackToBackgroundState(t *testing.T) {
	client := &mockAuditClient{}
	svc, err := NewService("emp-1", desktop.NewState(), setupOverrideTestRules(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	_ = batch.Add(events.WindowEvent{
		Timestamp:   now,
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	_ = batch.Add(events.OCREvent{
		Timestamp:  now.Add(5 * time.Millisecond),
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "FW: Project Notice",
				Box:  geometry.Rectangle{X: 100, Y: 100, Width: 200, Height: 30},
			},
		},
	})

	// Mouse click without ProcessName or WindowTitle set
	_ = batch.Add(events.MouseEvent{
		Timestamp: now.Add(10 * time.Millisecond),
		Action:    "click",
		Button:    "left",
		Position:  geometry.Point{X: 150, Y: 110},
	})

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 1 {
		t.Fatalf("expected 1 popup from background process fallback, got %d", len(popups))
	}
	if popups[0].Rule != "forwarded-email" || popups[0].Body != "Opened: FW: Project Notice" {
		t.Errorf("unexpected popup: %+v", popups[0])
	}
}

func TestService_ProcessTick_ClickChromeLegacyWindowFallback(t *testing.T) {
	client := &mockAuditClient{}
	svc, err := NewService("emp-1", desktop.NewState(), setupOverrideTestRules(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	batch, _ := events.NewTickBatch(0, now, now.Add(time.Second))

	_ = batch.Add(events.WindowEvent{
		Timestamp:   now,
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	_ = batch.Add(events.OCREvent{
		Timestamp:  now.Add(5 * time.Millisecond),
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "FW: Important Notice",
				Box:  geometry.Rectangle{X: 100, Y: 100, Width: 200, Height: 30},
			},
		},
	})

	_ = batch.Add(events.MouseEvent{
		Timestamp:   now.Add(10 * time.Millisecond),
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 150, Y: 110},
		ProcessName: "msedgewebview2.exe",
		WindowTitle: "Chrome Legacy Window",
	})

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}

	popups := client.Popups()
	if len(popups) != 1 {
		t.Fatalf("expected 1 popup, got %d", len(popups))
	}
	if popups[0].Rule != "forwarded-email" || popups[0].Body != "Opened: FW: Important Notice" {
		t.Errorf("unexpected popup: %+v", popups[0])
	}
}

func createStressBatch(workerID, i int) events.TickBatch {
	baseTime := time.Date(2026, 3, 10, 10, workerID, i, 0, time.UTC)
	batch, _ := events.NewTickBatch(i, baseTime, baseTime.Add(time.Second))

	_ = batch.Add(events.OCREvent{
		Timestamp:  baseTime.Add(10 * time.Millisecond),
		DisplayID:  0,
		Resolution: geometry.Size{Width: 1920, Height: 1080},
		Blocks: []display.OCRTextBlock{
			{
				Text: "FW: Multi-threaded Notice",
				Box:  geometry.Rectangle{X: 100, Y: 100, Width: 300, Height: 40},
			},
		},
	})

	_ = batch.Add(events.WindowEvent{
		Timestamp:   baseTime.Add(20 * time.Millisecond),
		ProcessName: "notepad.exe",
		WindowTitle: "Notepad Document",
	})

	_ = batch.Add(events.MouseEvent{
		Timestamp:   baseTime.Add(50 * time.Millisecond),
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 150, Y: 110},
		ProcessName: "OUTLOOK.EXE",
		WindowTitle: "Inbox - Outlook",
	})

	_ = batch.Add(events.MouseEvent{
		Timestamp:   baseTime.Add(60 * time.Millisecond),
		Action:      "click",
		Button:      "left",
		Position:    geometry.Point{X: 9999, Y: 9999},
		ProcessName: "OUTLOOK.EXE",
	})

	return batch
}

func TestService_ProcessTick_StressRapidBurstChaosAndRace(t *testing.T) {
	client := &mockAuditClient{}
	svc, err := NewService("emp-stress", desktop.NewState(), setupOverrideTestRules(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	numWorkers := 10
	numTicksPerWorker := 30

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < numTicksPerWorker; i++ {
				batch := createStressBatch(workerID, i)
				if err := svc.ProcessTick(context.Background(), batch); err != nil {
					t.Errorf("worker %d tick %d failed: %v", workerID, i, err)
				}
			}
		}(w)
	}

	wg.Wait()

	popups := client.Popups()
	expectedCount := numWorkers * numTicksPerWorker
	if len(popups) != expectedCount {
		t.Errorf("expected %d total popups from stress test, got %d", expectedCount, len(popups))
	}
}

func runWindowStep(t *testing.T, svc *Service, client *mockAuditClient, idx int, ts time.Time, proc, title string, wantPopups int) {
	t.Helper()
	batch, _ := events.NewTickBatch(idx, ts, ts.Add(time.Second))
	_ = batch.Add(events.WindowEvent{
		Timestamp:   ts,
		ProcessName: proc,
		WindowTitle: title,
	})
	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}
	if len(client.Popups()) != wantPopups {
		t.Fatalf("expected %d popups, got %d", wantPopups, len(client.Popups()))
	}
}

func runClipboardStep(t *testing.T, svc *Service, client *mockAuditClient, idx int, ts time.Time, text string, wantPopups int) {
	t.Helper()
	batch, _ := events.NewTickBatch(idx, ts, ts.Add(time.Second))
	_ = batch.Add(events.ClipboardEvent{
		Timestamp: ts,
		Action:    "copy",
		Text:      text,
	})
	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed: %v", err)
	}
	if len(client.Popups()) != wantPopups {
		t.Fatalf("expected %d popups, got %d", wantPopups, len(client.Popups()))
	}
}

func TestService_ContextCycling_Window(t *testing.T) {
	client := &mockAuditClient{}
	procPat, _ := rules.NewPatternList("OUTLOOK.EXE")
	tpl, _ := rules.NewPopupTemplate("Outlook Focus", "Active: {process}")
	r1, err := rules.NewRule("outlook-focus", rules.WhenConditions{
		Process: &procPat,
	}, tpl)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	svc, err := NewService("emp-1", desktop.NewState(), []rules.Rule{r1}, client)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)

	// 1. Initial focus triggers rule
	runWindowStep(t, svc, client, 0, now, "OUTLOOK.EXE", "Inbox - Outlook", 1)
	// 2. Same window focus is deduplicated
	runWindowStep(t, svc, client, 1, now.Add(time.Second), "OUTLOOK.EXE", "Inbox - Outlook", 1)
	// 3. Switching to unmonitored window emits no popup
	runWindowStep(t, svc, client, 2, now.Add(2*time.Second), "notepad.exe", "Untitled - Notepad", 1)
	// 4. Returning to monitored window re-triggers rule
	runWindowStep(t, svc, client, 3, now.Add(3*time.Second), "OUTLOOK.EXE", "Inbox - Outlook", 2)
}

func TestService_ContextCycling_Clipboard(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, err := NewService("emp-1", desktop.NewState(), rls, client)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	runWindowStep(t, svc, client, 0, now, "OUTLOOK.EXE", "Inbox - Outlook", 0)

	// 1. Initial invoice copy triggers rule
	runClipboardStep(t, svc, client, 1, now.Add(time.Second), "INV-100", 1)
	// 2. Duplicate copy is deduplicated
	runClipboardStep(t, svc, client, 2, now.Add(2*time.Second), "INV-100", 1)
	// 3. Copying regular text resets clipboard state without popup
	runClipboardStep(t, svc, client, 3, now.Add(3*time.Second), "Hello World", 1)
	// 4. Copying invoice again re-triggers rule
	runClipboardStep(t, svc, client, 4, now.Add(4*time.Second), "INV-100", 2)
}

func TestService_ContextFlappingTorture(t *testing.T) {
	client := &mockAuditClient{}
	rls := sampleTestRules(t)
	svc, err := NewService("emp-torture", desktop.NewState(), rls, client)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	apps := []struct {
		proc  string
		title string
	}{
		{"OUTLOOK.EXE", "Inbox - Outlook"},
		{"chrome.exe", "Google - Chrome"},
		{"EXCEL.EXE", "Book1 - Excel"},
		{"notepad.exe", "Untitled - Notepad"},
		{"", ""},
		{"unknown.exe", strings.Repeat("Huge Title ", 100)},
	}

	batch, _ := events.NewTickBatch(0, now, now.Add(10*time.Second))

	for i := 0; i < 100; i++ {
		app := apps[i%len(apps)]
		ts := now.Add(time.Duration(i*10) * time.Millisecond)

		_ = batch.Add(events.WindowEvent{
			Timestamp:   ts,
			ProcessName: app.proc,
			WindowTitle: app.title,
		})

		switch i {
		case 10:
			_ = batch.Add(events.ClipboardEvent{
				Timestamp: ts.Add(time.Millisecond),
				Text:      "INV-999",
			})
		case 30:
			_ = batch.Add(events.ClipboardEvent{
				Timestamp: ts.Add(time.Millisecond),
				Text:      strings.Repeat("X", 50000),
			})
		case 50:
			_ = batch.Add(events.ClipboardEvent{
				Timestamp: ts.Add(time.Millisecond),
				Text:      "",
			})
		case 70:
			_ = batch.Add(events.ClipboardEvent{
				Timestamp: ts.Add(time.Millisecond),
				Text:      "INV-999",
			})
		}
	}

	if err := svc.ProcessTick(context.Background(), batch); err != nil {
		t.Fatalf("ProcessTick failed during flapping torture: %v", err)
	}
}
