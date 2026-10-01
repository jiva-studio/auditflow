package server_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

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

type serverResponseDTO struct {
	Total  int             `json:"total"`
	Popups []expectedPopup `json:"popups"`
}

func loadFixture(t *testing.T, empID string) agentFixture {
	t.Helper()
	root := testutil.FindProjectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "tests", "e2e", "agent", "fixtures", empID+".json"))
	if err != nil {
		t.Fatalf("failed to read fixture for %s: %v", empID, err)
	}
	var f agentFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("failed to parse fixture for %s: %v", empID, err)
	}
	return f
}

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForHealth(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("health check failed for %s after %v", url, timeout)
}

func startProcess(t *testing.T, bin string, env map[string]string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(bin)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start %s: %v", bin, err)
	}
	return cmd
}

func stopProcess(t *testing.T, cmd *exec.Cmd) {
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

func queryServer(t *testing.T, serverURL, queryPath string) serverResponseDTO {
	t.Helper()
	resp, err := http.Get(serverURL + queryPath)
	if err != nil {
		t.Fatalf("failed to query server: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var dto serverResponseDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("failed to parse JSON response (%s): %v", string(body), err)
	}
	return dto
}

func TestServer_E2E_FullPipeline(t *testing.T) {
	serverBin := testutil.GetServerBin(t)
	agentBin := testutil.GetAgentBin(t)
	streamerBin := testutil.GetStreamerBin(t)
	rulesPath := testutil.GetRulesPath(t)
	dataDir := testutil.GetDataDir(t)

	serverPort := getFreePort(t)
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", serverPort)

	// 1. Start real Central Audit Server
	serverCmd := startProcess(t, serverBin, map[string]string{
		"PORT": fmt.Sprintf("%d", serverPort),
	})
	defer stopProcess(t, serverCmd)

	waitForHealth(t, serverURL, 5*time.Second)

	// 2. Stream all 5 recording scenarios into dedicated agents pointing to the server
	employees := []string{"emp-1", "emp-2", "emp-3", "emp-synthetic", "emp-edge"}

	for _, empID := range employees {
		archivePath := filepath.Join(dataDir, empID+".tar.gz")
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			t.Fatalf("recording %s not found", archivePath)
		}

		agentPort := getFreePort(t)
		agentURL := fmt.Sprintf("http://127.0.0.1:%d", agentPort)

		agentCmd := startProcess(t, agentBin, map[string]string{
			"PORT":        fmt.Sprintf("%d", agentPort),
			"RULES_PATH":  rulesPath,
			"SERVER_URL":  serverURL,
			"EMPLOYEE_ID": empID,
		})

		waitForHealth(t, agentURL, 5*time.Second)

		out, err := testutil.RunStreamer(t, streamerBin, archivePath, agentURL)
		if err != nil {
			stopProcess(t, agentCmd)
			t.Fatalf("streamer failed for %s: %v\nOutput: %s", empID, err, string(out))
		}

		time.Sleep(150 * time.Millisecond)
		stopProcess(t, agentCmd)
	}

	time.Sleep(200 * time.Millisecond)

	// 3. Query and verify results for each employee
	totalExpected := 0
	for _, empID := range employees {
		fixture := loadFixture(t, empID)
		totalExpected += fixture.TotalPopups

		dto := queryServer(t, serverURL, fmt.Sprintf("/audit?employee=%s", empID))
		if dto.Total != fixture.TotalPopups {
			t.Errorf("[%s] expected %d popups, got %d", empID, fixture.TotalPopups, dto.Total)
		}

		for i, exp := range fixture.Popups {
			if i >= len(dto.Popups) {
				t.Fatalf("[%s] missing expected popup #%d", empID, i+1)
			}
			act := dto.Popups[i]
			if act.Employee != exp.Employee || act.Rule != exp.Rule || act.Title != exp.Title || act.Body != exp.Body {
				t.Errorf("[%s] popup #%d mismatch:\n  got:  %+v\n  want: %+v", empID, i+1, act, exp)
			}
		}
	}

	// 4. Query total popups across all employees
	allDTO := queryServer(t, serverURL, "/audit")
	t.Logf("Total recorded popups on Central Server: %d (expected %d)", allDTO.Total, totalExpected)
	if allDTO.Total != totalExpected {
		t.Fatalf("expected %d total popups, got %d", totalExpected, allDTO.Total)
	}

	// 5. Query by specific rule
	invoiceDTO := queryServer(t, serverURL, "/audit?rule=invoice-ready-to-attach")
	t.Logf("Total 'invoice-ready-to-attach' popups on Central Server: %d", invoiceDTO.Total)
	if invoiceDTO.Total == 0 {
		t.Fatal("expected non-zero popups for invoice-ready-to-attach rule")
	}
	for _, p := range invoiceDTO.Popups {
		if p.Rule != "invoice-ready-to-attach" {
			t.Errorf("expected rule invoice-ready-to-attach, got %s", p.Rule)
		}
	}
}
