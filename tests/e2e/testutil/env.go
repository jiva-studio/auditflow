package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// FindProjectRoot locates the root assessment directory containing .git
func FindProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("project root not found (.git missing in parent hierarchy)")
		}
		dir = parent
	}
}

// GetStreamerBin returns the path to the compiled streamer binary
func GetStreamerBin(t *testing.T) string {
	t.Helper()
	binPath := os.Getenv("STREAMER_BIN")
	if binPath != "" {
		return binPath
	}
	return filepath.Join(FindProjectRoot(t), "bin", "streamer")
}

// GetAgentBin returns the path to the compiled agent binary
func GetAgentBin(t *testing.T) string {
	t.Helper()
	binPath := os.Getenv("AGENT_BIN")
	if binPath != "" {
		return binPath
	}
	return filepath.Join(FindProjectRoot(t), "bin", "agent")
}

// GetRulesPath returns the path to data/rules.json
func GetRulesPath(t *testing.T) string {
	t.Helper()
	rulesPath := os.Getenv("RULES_PATH")
	if rulesPath != "" {
		return rulesPath
	}
	return filepath.Join(FindProjectRoot(t), "data", "rules.json")
}

// GetDataDir returns the path to real test recordings
func GetDataDir(t *testing.T) string {
	t.Helper()
	dataDir := os.Getenv("DATA_DIR")
	if dataDir != "" {
		return dataDir
	}
	return filepath.Join(FindProjectRoot(t), "data")
}

// NewMockAgentServer creates a test HTTP server receiving protobuf tick batches
func NewMockAgentServer(t *testing.T, onBatch func(body []byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-protobuf" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if onBatch != nil {
			onBatch(body)
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// NewMockCentralServer creates a test HTTP server receiving protobuf audit popups
func NewMockCentralServer(t *testing.T, onPopup func(body []byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/audit" || r.Header.Get("Content-Type") != "application/x-protobuf" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if onPopup != nil {
			onPopup(body)
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// RunStreamer runs the streamer binary with default and optional extra env vars
func RunStreamer(t *testing.T, binPath, archivePath, agentURL string, extraEnv ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"RECORDING_PATH="+archivePath,
		"AGENT_URL="+agentURL,
		"TICK_MS=0",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	return cmd.CombinedOutput()
}

// GetEnvInt parses an integer from env var or returns defaultValue
func GetEnvInt(key string, defaultValue int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val <= 0 {
		return defaultValue
	}
	return val
}

// GetEnvFloat parses a float64 from env var or returns defaultValue
func GetEnvFloat(key string, defaultValue float64) float64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil || val <= 0 {
		return defaultValue
	}
	return val
}

// GetEnvString returns string from env var or defaultValue
func GetEnvString(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
