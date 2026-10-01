package agent_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	v1 "assessment/modules/libs/protocol/gen/go/v1"
	"assessment/tests/e2e/testutil"
)

type expectedPopup struct {
	Employee string `json:"employee"`
	Rule     string `json:"rule"`
	TS       string `json:"ts"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

type agentFixture struct {
	EmployeeID  string          `json:"employee_id"`
	TotalPopups int             `json:"total_popups"`
	Popups      []expectedPopup `json:"popups"`
}

func loadAgentFixture(t *testing.T, fixtureName string) agentFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("fixtures", fixtureName))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixtureName, err)
	}
	var f agentFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("failed to parse fixture %s: %v", fixtureName, err)
	}
	return f
}

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForHealth(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 200 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("agent at %s failed to become healthy within %v", url, timeout)
}

func startAgent(t *testing.T, binPath, rulesPath, serverURL, employeeID string, port int) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		"RULES_PATH="+rulesPath,
		"SERVER_URL="+serverURL,
		"EMPLOYEE_ID="+employeeID,
		"SERVER_TIMEOUT_MS=5000",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start agent process: %v", err)
	}
	return cmd
}

func stopAgent(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
	}
}

func TestAgent_E2E_AllEmployees(t *testing.T) {
	agentBin := testutil.GetAgentBin(t)
	streamerBin := testutil.GetStreamerBin(t)
	rulesPath := testutil.GetRulesPath(t)
	dataDir := testutil.GetDataDir(t)

	employees := []string{"emp-1", "emp-2", "emp-3", "emp-synthetic", "emp-edge", "emp-multidisplay", "emp-quadhd", "emp-lefthand", "emp-placeholders", "emp-clickprocess", "emp-contextreset"}

	for _, empID := range employees {
		empID := empID
		t.Run(empID, func(t *testing.T) {
			archivePath := filepath.Join(dataDir, empID+".tar.gz")
			if _, err := os.Stat(archivePath); os.IsNotExist(err) {
				t.Skipf("recording %s not found, skipping", archivePath)
			}

			fixture := loadAgentFixture(t, empID+".json")

			var mu sync.Mutex
			var receivedPopups []*v1.Popup

			mockServer := testutil.NewMockCentralServer(t, func(body []byte) {
				var p v1.Popup
				if err := proto.Unmarshal(body, &p); err == nil {
					mu.Lock()
					receivedPopups = append(receivedPopups, &p)
					mu.Unlock()
				}
			})
			defer mockServer.Close()

			port := getFreePort(t)
			agentURL := fmt.Sprintf("http://127.0.0.1:%d", port)

			agentCmd := startAgent(t, agentBin, rulesPath, mockServer.URL, empID, port)
			defer stopAgent(t, agentCmd)

			waitForHealth(t, agentURL, 5*time.Second)

			out, err := testutil.RunStreamer(t, streamerBin, archivePath, agentURL)
			if err != nil {
				t.Fatalf("streamer replay failed: %v, output:\n%s", err, string(out))
			}

			time.Sleep(200 * time.Millisecond)

			mu.Lock()
			defer mu.Unlock()

			t.Logf("[%s] received %d popups:", empID, len(receivedPopups))
			for i, p := range receivedPopups {
				t.Logf("  popup #%d: rule=%s, ts=%s, title=%q, body=%q",
					i+1, p.GetRule(), p.GetTs().AsTime().Format(time.RFC3339), p.GetTitle(), p.GetBody())
			}

			if len(receivedPopups) != fixture.TotalPopups {
				t.Fatalf("[%s] expected %d popups, got %d", empID, fixture.TotalPopups, len(receivedPopups))
			}

			for i, exp := range fixture.Popups {
				act := receivedPopups[i]
				if act.GetEmployee() != exp.Employee {
					t.Errorf("popup #%d employee mismatch: got %s, want %s", i+1, act.GetEmployee(), exp.Employee)
				}
				if act.GetRule() != exp.Rule {
					t.Errorf("popup #%d rule mismatch: got %s, want %s", i+1, act.GetRule(), exp.Rule)
				}
				if act.GetTitle() != exp.Title {
					t.Errorf("popup #%d title mismatch: got %q, want %q", i+1, act.GetTitle(), exp.Title)
				}
				if act.GetBody() != exp.Body {
					t.Errorf("popup #%d body mismatch: got %q, want %q", i+1, act.GetBody(), exp.Body)
				}
				if act.GetTs().AsTime().Format(time.RFC3339) != exp.TS {
					t.Errorf("popup #%d ts mismatch: got %s, want %s", i+1, act.GetTs().AsTime().Format(time.RFC3339), exp.TS)
				}
			}
		})
	}
}

func TestAgent_E2E_LeftHandedMouseClicks(t *testing.T) {
	agentBin := testutil.GetAgentBin(t)
	streamerBin := testutil.GetStreamerBin(t)
	rulesPath := testutil.GetRulesPath(t)

	archivePath := filepath.Join(t.TempDir(), "emp-lefthanded.tar.gz")
	if err := testutil.GenerateLeftHandedMouseRecording(t, archivePath); err != nil {
		t.Fatalf("failed to generate left-handed mouse recording: %v", err)
	}

	var mu sync.Mutex
	var receivedPopups []*v1.Popup

	mockServer := testutil.NewMockCentralServer(t, func(body []byte) {
		var p v1.Popup
		if err := proto.Unmarshal(body, &p); err == nil {
			mu.Lock()
			receivedPopups = append(receivedPopups, &p)
			mu.Unlock()
		}
	})
	defer mockServer.Close()

	port := getFreePort(t)
	agentURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	empID := "emp-lefthanded"
	agentCmd := startAgent(t, agentBin, rulesPath, mockServer.URL, empID, port)
	defer stopAgent(t, agentCmd)

	waitForHealth(t, agentURL, 5*time.Second)

	out, err := testutil.RunStreamer(t, streamerBin, archivePath, agentURL)
	if err != nil {
		t.Fatalf("streamer replay failed: %v, output:\n%s", err, string(out))
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	expectedPopups := []expectedPopup{
		{
			Employee: "emp-lefthanded",
			Rule:     "forwarded-email-opened",
			TS:       "2026-03-10T10:00:01Z",
			Title:    "Forwarded email",
			Body:     "You opened a forwarded email: FW: Left Handed Budget Update",
		},
		{
			Employee: "emp-lefthanded",
			Rule:     "jira-ticket-done",
			TS:       "2026-03-10T10:00:02Z",
			Title:    "Ticket moved to Done",
			Body:     "DEV-42 - Jira - Google Chrome",
		},
		{
			Employee: "emp-lefthanded",
			Rule:     "urgent-email-opened",
			TS:       "2026-03-10T10:00:03Z",
			Title:    "Urgent email",
			Body:     "Urgent Security Patch Review",
		},
	}

	if len(receivedPopups) != len(expectedPopups) {
		t.Fatalf("expected %d popups, got %d", len(expectedPopups), len(receivedPopups))
	}

	for i, exp := range expectedPopups {
		act := receivedPopups[i]
		if act.GetEmployee() != exp.Employee {
			t.Errorf("popup #%d employee mismatch: got %s, want %s", i+1, act.GetEmployee(), exp.Employee)
		}
		if act.GetRule() != exp.Rule {
			t.Errorf("popup #%d rule mismatch: got %s, want %s", i+1, act.GetRule(), exp.Rule)
		}
		if act.GetTitle() != exp.Title {
			t.Errorf("popup #%d title mismatch: got %q, want %q", i+1, act.GetTitle(), exp.Title)
		}
		if act.GetBody() != exp.Body {
			t.Errorf("popup #%d body mismatch: got %q, want %q", i+1, act.GetBody(), exp.Body)
		}
		if act.GetTs().AsTime().Format(time.RFC3339) != exp.TS {
			t.Errorf("popup #%d ts mismatch: got %s, want %s", i+1, act.GetTs().AsTime().Format(time.RFC3339), exp.TS)
		}
	}
}

func TestAgent_E2E_TemplatePlaceholderCleanup(t *testing.T) {
	agentBin := testutil.GetAgentBin(t)
	streamerBin := testutil.GetStreamerBin(t)

	// Create temporary rules configuration with unresolved placeholders in templates
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "rules.json")
	rulesContent := `{
		"rules": [
			{
				"id": "forwarded-email-placeholder-clean",
				"when": { "click": "FW:*", "process": ["OUTLOOK.EXE", "olk.exe"] },
				"popup": {
					"title": "Forwarded email [{unresolved_tag}]",
					"body": "Opened: {click} on {window_title} (missing: {unresolved_meta}, code: {missing_code})"
				}
			}
		]
	}`
	if err := os.WriteFile(rulesFile, []byte(rulesContent), 0644); err != nil {
		t.Fatalf("failed to write custom rules file: %v", err)
	}

	empID := "emp-placeholder-test"
	archivePath := filepath.Join(tmpDir, empID+".tar.gz")
	if _, err := testutil.GenerateTemplatePlaceholderRecording(t, archivePath, empID); err != nil {
		t.Fatalf("failed to generate synthetic recording: %v", err)
	}

	var mu sync.Mutex
	var receivedPopups []*v1.Popup

	mockServer := testutil.NewMockCentralServer(t, func(body []byte) {
		var p v1.Popup
		if err := proto.Unmarshal(body, &p); err == nil {
			mu.Lock()
			receivedPopups = append(receivedPopups, &p)
			mu.Unlock()
		}
	})
	defer mockServer.Close()

	port := getFreePort(t)
	agentURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	agentCmd := startAgent(t, agentBin, rulesFile, mockServer.URL, empID, port)
	defer stopAgent(t, agentCmd)

	waitForHealth(t, agentURL, 5*time.Second)

	out, err := testutil.RunStreamer(t, streamerBin, archivePath, agentURL)
	if err != nil {
		t.Fatalf("streamer replay failed: %v, output:\n%s", err, string(out))
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedPopups) != 1 {
		t.Fatalf("expected 1 popup, got %d", len(receivedPopups))
	}

	popup := receivedPopups[0]
	if popup.GetEmployee() != empID {
		t.Errorf("employee mismatch: got %q, want %q", popup.GetEmployee(), empID)
	}
	if popup.GetRule() != "forwarded-email-placeholder-clean" {
		t.Errorf("rule mismatch: got %q, want %q", popup.GetRule(), "forwarded-email-placeholder-clean")
	}

	expectedTitle := "Forwarded email []"
	expectedBody := "Opened: FW: Project Alpha Update on Inbox - Outlook (missing: , code: )"

	if popup.GetTitle() != expectedTitle {
		t.Errorf("title mismatch: got %q, want %q", popup.GetTitle(), expectedTitle)
	}
	if popup.GetBody() != expectedBody {
		t.Errorf("body mismatch: got %q, want %q", popup.GetBody(), expectedBody)
	}
}

func TestAgent_E2E_ClickProcessContext(t *testing.T) {
	agentBin := testutil.GetAgentBin(t)
	streamerBin := testutil.GetStreamerBin(t)
	rulesPath := testutil.GetRulesPath(t)

	empID := "emp-click-context"
	tmpArchive := filepath.Join(t.TempDir(), "click_context.tar.gz")

	_, err := testutil.GenerateClickProcessContextRecording(t, tmpArchive, empID)
	if err != nil {
		t.Fatalf("failed to generate click process context recording: %v", err)
	}

	var mu sync.Mutex
	var receivedPopups []*v1.Popup

	mockServer := testutil.NewMockCentralServer(t, func(body []byte) {
		var p v1.Popup
		if err := proto.Unmarshal(body, &p); err == nil {
			mu.Lock()
			receivedPopups = append(receivedPopups, &p)
			mu.Unlock()
		}
	})
	defer mockServer.Close()

	port := getFreePort(t)
	agentURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	agentCmd := startAgent(t, agentBin, rulesPath, mockServer.URL, empID, port)
	defer stopAgent(t, agentCmd)

	waitForHealth(t, agentURL, 5*time.Second)

	out, err := testutil.RunStreamer(t, streamerBin, tmpArchive, agentURL)
	if err != nil {
		t.Fatalf("streamer replay failed: %v, output:\n%s", err, string(out))
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	expectedPopups := []expectedPopup{
		{
			Employee: empID,
			Rule:     "forwarded-email-opened",
			Title:    "Forwarded email",
			Body:     "You opened a forwarded email: FW: Q3 Audit Report",
		},
		{
			Employee: empID,
			Rule:     "jira-ticket-done",
			Title:    "Ticket moved to Done",
			Body:     "PROJ-101 - Jira - Dashboard",
		},
	}

	if len(receivedPopups) != len(expectedPopups) {
		t.Fatalf("expected %d popups, got %d", len(expectedPopups), len(receivedPopups))
	}

	for i, exp := range expectedPopups {
		act := receivedPopups[i]
		if act.GetEmployee() != exp.Employee {
			t.Errorf("popup #%d employee mismatch: got %s, want %s", i+1, act.GetEmployee(), exp.Employee)
		}
		if act.GetRule() != exp.Rule {
			t.Errorf("popup #%d rule mismatch: got %s, want %s", i+1, act.GetRule(), exp.Rule)
		}
		if act.GetTitle() != exp.Title {
			t.Errorf("popup #%d title mismatch: got %q, want %q", i+1, act.GetTitle(), exp.Title)
		}
		if act.GetBody() != exp.Body {
			t.Errorf("popup #%d body mismatch: got %q, want %q", i+1, act.GetBody(), exp.Body)
		}
	}
}

func TestAgent_E2E_ContextReset(t *testing.T) {
	agentBin := testutil.GetAgentBin(t)
	streamerBin := testutil.GetStreamerBin(t)
	rulesPath := testutil.GetRulesPath(t)

	empID := "emp-context-reset"
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "context_reset.tar.gz")

	if err := testutil.GenerateContextResetRecording(t, archivePath, empID); err != nil {
		t.Fatalf("failed to generate context reset recording: %v", err)
	}

	var mu sync.Mutex
	var receivedPopups []*v1.Popup

	mockServer := testutil.NewMockCentralServer(t, func(body []byte) {
		var p v1.Popup
		if err := proto.Unmarshal(body, &p); err == nil {
			mu.Lock()
			receivedPopups = append(receivedPopups, &p)
			mu.Unlock()
		}
	})
	defer mockServer.Close()

	port := getFreePort(t)
	agentURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	agentCmd := startAgent(t, agentBin, rulesPath, mockServer.URL, empID, port)
	defer stopAgent(t, agentCmd)

	waitForHealth(t, agentURL, 5*time.Second)

	out, err := testutil.RunStreamer(t, streamerBin, archivePath, agentURL)
	if err != nil {
		t.Fatalf("streamer replay failed: %v, output:\n%s", err, string(out))
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	t.Logf("[%s] received %d popups:", empID, len(receivedPopups))
	for i, p := range receivedPopups {
		t.Logf("  popup #%d: rule=%s, ts=%s, title=%q, body=%q",
			i+1, p.GetRule(), p.GetTs().AsTime().Format(time.RFC3339), p.GetTitle(), p.GetBody())
	}

	if len(receivedPopups) != 3 {
		t.Fatalf("[%s] expected 3 popups across context resets, got %d", empID, len(receivedPopups))
	}

	for i, p := range receivedPopups {
		if p.GetEmployee() != empID {
			t.Errorf("popup #%d employee mismatch: got %s, want %s", i+1, p.GetEmployee(), empID)
		}
		if p.GetRule() != "invoice-ready-to-attach" {
			t.Errorf("popup #%d rule mismatch: got %s, want invoice-ready-to-attach", i+1, p.GetRule())
		}
		if p.GetTitle() != "Invoice in clipboard" {
			t.Errorf("popup #%d title mismatch: got %q, want %q", i+1, p.GetTitle(), "Invoice in clipboard")
		}
		if p.GetBody() != "You copied INV-100. Attach the invoice to this email?" {
			t.Errorf("popup #%d body mismatch: got %q, want %q", i+1, p.GetBody(), "You copied INV-100. Attach the invoice to this email?")
		}
	}
}
