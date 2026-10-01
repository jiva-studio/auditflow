package service

import (
	"context"
	"errors"
	"fmt"
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
