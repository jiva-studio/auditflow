// Package main is the entrypoint for the streamer service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/client"
	"assessment/modules/apps/streamer/internal/adapters/reporter"
	"assessment/modules/apps/streamer/internal/adapters/source"
	"assessment/modules/apps/streamer/internal/adapters/timer"
	"assessment/modules/apps/streamer/internal/service"
	"assessment/modules/libs/telemetry"
)

var exitHandler = func(format string, args ...any) {
	slog.Default().Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func main() {
	logger := telemetry.NewLogger(telemetry.LoggerConfig{
		ServiceName: "streamer",
		Level:       slog.LevelInfo,
		Format:      telemetry.FormatJSON,
	})
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		exitHandler("[streamer] error: %v", err)
	}
}

type streamerConfig struct {
	recordingPath string
	agentURL      string
	tickInterval  time.Duration
}

func loadStreamerConfig() (streamerConfig, error) {
	recordingPath := getEnv("RECORDING_PATH", "/data/recording.tar.gz")
	agentURL := getEnv("AGENT_URL", "http://agent:8081")
	tickMSStr := getEnv("TICK_MS", "1000")

	tickMS, err := strconv.Atoi(tickMSStr)
	if err != nil {
		return streamerConfig{}, err
	}
	if tickMS < 0 {
		return streamerConfig{}, strconv.ErrRange
	}

	return streamerConfig{
		recordingPath: recordingPath,
		agentURL:      agentURL,
		tickInterval:  time.Duration(tickMS) * time.Millisecond,
	}, nil
}

func run(loggers ...*slog.Logger) error {
	var logger *slog.Logger
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	} else {
		logger = slog.Default()
	}

	cfg, err := loadStreamerConfig()
	if err != nil {
		return err
	}

	traceID := telemetry.GenerateTraceID()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ctx = telemetry.WithTraceID(ctx, traceID)
	ctx = telemetry.WithLogger(ctx, logger.With(slog.String("trace_id", traceID)))

	logger.InfoContext(ctx, "starting replay",
		slog.String("recording_path", cfg.recordingPath),
		slog.String("agent_url", cfg.agentURL),
		slog.Duration("tick_interval", cfg.tickInterval),
		slog.String("trace_id", traceID),
	)

	errReporter := reporter.NewLogReporter(os.Stderr)
	eventSource := source.NewTarGzEventSource(cfg.recordingPath, errReporter)
	agentClient, err := client.NewHTTPClient(cfg.agentURL, 30*time.Second)
	if err != nil {
		return err
	}

	realTimer := timer.NewRealTimer()
	replaySvc, err := service.NewReplayService(eventSource, agentClient, realTimer, cfg.tickInterval)
	if err != nil {
		return err
	}

	meta, count, err := replaySvc.Run(ctx)
	if err != nil {
		if ctx.Err() != nil {
			logger.InfoContext(ctx, "replay cancelled after interrupt signal",
				slog.Int("dispatched_ticks", count),
				slog.String("trace_id", traceID),
			)
			return nil
		}
		return err
	}

	logger.InfoContext(ctx, "replay finished successfully",
		slog.String("employee_id", meta.EmployeeID),
		slog.String("session_id", meta.SessionID),
		slog.Int("dispatched_ticks", count),
		slog.String("trace_id", traceID),
	)
	return nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}
