// Package main is the entrypoint for the runner service that coordinates replaying archives into agents.
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"assessment/modules/libs/telemetry"
)

var exitHandler = func(format string, args ...any) {
	slog.Default().Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

type runnerConfig struct {
	serverURL   string
	rulesPath   string
	dataPattern string
	agentBin    string
	streamerBin string
	concurrency int
	tickMS      string
}

func main() {
	logger := telemetry.NewLogger(telemetry.LoggerConfig{
		ServiceName: "runner",
		Level:       slog.LevelInfo,
		Format:      telemetry.FormatJSON,
	})
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		exitHandler("[runner] fatal error: %v", err)
	}
}

func loadRunnerConfig() (runnerConfig, error) {
	serverURL := getEnv("SERVER_URL", "http://server:8080")
	rulesPath := getEnv("RULES_PATH", "/rules.json")
	dataDir := getEnv("DATA_DIR", "/data")
	dataPattern := getEnv("DATA_PATTERN", filepath.Join(dataDir, "*.tar.gz"))
	agentBin := getEnv("AGENT_BIN", "/app/agent")
	streamerBin := getEnv("STREAMER_BIN", "/app/streamer")
	tickMS := getEnv("TICK_MS", "10")

	// If default binary paths don't exist, search in PATH or local ./bin
	if _, err := os.Stat(agentBin); os.IsNotExist(err) {
		if path, err := exec.LookPath("agent"); err == nil {
			agentBin = path
		} else if _, err := os.Stat("./bin/agent"); err == nil {
			agentBin = "./bin/agent"
		}
	}
	if _, err := os.Stat(streamerBin); os.IsNotExist(err) {
		if path, err := exec.LookPath("streamer"); err == nil {
			streamerBin = path
		} else if _, err := os.Stat("./bin/streamer"); err == nil {
			streamerBin = "./bin/streamer"
		}
	}

	concurrencyStr := getEnv("CONCURRENCY", "10")
	concurrency, err := strconv.Atoi(concurrencyStr)
	if err != nil || concurrency <= 0 {
		return runnerConfig{}, fmt.Errorf("invalid CONCURRENCY %q: must be positive integer", concurrencyStr)
	}

	return runnerConfig{
		serverURL:   serverURL,
		rulesPath:   rulesPath,
		dataPattern: dataPattern,
		agentBin:    agentBin,
		streamerBin: streamerBin,
		concurrency: concurrency,
		tickMS:      tickMS,
	}, nil
}

func run(logger *slog.Logger) error {
	cfg, err := loadRunnerConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	archives, err := filepath.Glob(cfg.dataPattern)
	if err != nil {
		return fmt.Errorf("scan data pattern %q: %w", cfg.dataPattern, err)
	}
	if len(archives) == 0 {
		return fmt.Errorf("no archive files matched pattern %q", cfg.dataPattern)
	}

	logger.InfoContext(ctx, "starting replay runner",
		slog.Int("total_archives", len(archives)),
		slog.Int("concurrency", cfg.concurrency),
		slog.String("server_url", cfg.serverURL),
		slog.String("rules_path", cfg.rulesPath),
	)

	return runWorkerPool(ctx, archives, cfg, logger)
}

func runWorkerPool(ctx context.Context, archives []string, cfg runnerConfig, logger *slog.Logger) error {
	jobs := make(chan string, len(archives))
	for _, a := range archives {
		jobs <- a
	}
	close(jobs)

	numWorkers := cfg.concurrency
	if numWorkers > len(archives) {
		numWorkers = len(archives)
	}

	var wg sync.WaitGroup
	var errorCount atomic.Int64
	var completedCount atomic.Int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for archivePath := range jobs {
				if ctx.Err() != nil {
					return
				}
				logger.InfoContext(ctx, "worker processing archive",
					slog.Int("worker_id", workerID),
					slog.String("archive", filepath.Base(archivePath)),
				)

				if err := processSingleArchive(ctx, archivePath, cfg, logger); err != nil {
					errorCount.Add(1)
					logger.ErrorContext(ctx, "failed processing archive",
						slog.String("archive", filepath.Base(archivePath)),
						slog.Any("error", err),
					)
				} else {
					completedCount.Add(1)
					logger.InfoContext(ctx, "successfully processed archive",
						slog.String("archive", filepath.Base(archivePath)),
					)
				}
			}
		}(i + 1)
	}

	wg.Wait()

	if ctx.Err() != nil {
		return fmt.Errorf("runner interrupted: %w", ctx.Err())
	}

	totalErrors := errorCount.Load()
	if totalErrors > 0 {
		return fmt.Errorf("%d of %d archives failed during replay", totalErrors, len(archives))
	}

	logger.InfoContext(ctx, "all archives replayed successfully",
		slog.Int64("completed", completedCount.Load()),
		slog.Int("total", len(archives)),
	)
	return nil
}

func processSingleArchive(ctx context.Context, archivePath string, cfg runnerConfig, logger *slog.Logger) error {
	empID, err := extractEmployeeID(archivePath)
	if err != nil {
		return fmt.Errorf("extract employee_id: %w", err)
	}

	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("get free port: %w", err)
	}

	agentCmd := exec.CommandContext(ctx, cfg.agentBin)
	agentCmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("RULES_PATH=%s", cfg.rulesPath),
		fmt.Sprintf("SERVER_URL=%s", cfg.serverURL),
		fmt.Sprintf("EMPLOYEE_ID=%s", empID),
	)
	agentCmd.Stdout = os.Stdout
	agentCmd.Stderr = os.Stderr

	if err := agentCmd.Start(); err != nil {
		return fmt.Errorf("start agent (%s): %w", empID, err)
	}

	defer func() {
		if agentCmd.Process != nil {
			_ = agentCmd.Process.Signal(syscall.SIGTERM)
			done := make(chan error, 1)
			go func() { done <- agentCmd.Wait() }()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = agentCmd.Process.Kill()
			}
		}
	}()

	agentURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := waitForReady(ctx, agentURL, 10*time.Second); err != nil {
		return fmt.Errorf("wait for agent ready: %w", err)
	}

	streamerCmd := exec.CommandContext(ctx, cfg.streamerBin)
	streamerCmd.Env = append(os.Environ(),
		fmt.Sprintf("RECORDING_PATH=%s", archivePath),
		fmt.Sprintf("AGENT_URL=%s", agentURL),
		fmt.Sprintf("TICK_MS=%s", cfg.tickMS),
	)
	streamerCmd.Stdout = os.Stdout
	streamerCmd.Stderr = os.Stderr

	if err := streamerCmd.Run(); err != nil {
		return fmt.Errorf("streamer execution failed for %s: %w", empID, err)
	}

	return nil
}

func extractEmployeeID(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar next: %w", err)
		}

		if filepath.Base(hdr.Name) == "metadata.json" {
			var meta struct {
				EmployeeID string `json:"employee_id"`
			}
			if err := json.NewDecoder(tr).Decode(&meta); err != nil {
				return "", fmt.Errorf("decode metadata.json: %w", err)
			}
			if meta.EmployeeID == "" {
				return "", errors.New("metadata.json contains empty employee_id")
			}
			return meta.EmployeeID, nil
		}
	}
	return "", errors.New("metadata.json not found in archive")
}

func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen free port: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForReady(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/health/ready", nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	return fmt.Errorf("readiness timeout exceeded for %s", url)
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}
