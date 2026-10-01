// Package main is the entrypoint for the streamer service.
package main

import (
	"context"
	"log"
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
)

var exitHandler = log.Fatalf

func main() {
	if err := run(); err != nil {
		exitHandler("[streamer] error: %v", err)
	}
}

func run() error {
	recordingPath := getEnv("RECORDING_PATH", "/data/recording.tar.gz")
	agentURL := getEnv("AGENT_URL", "http://agent:8081")
	tickMSStr := getEnv("TICK_MS", "1000")

	tickMS, err := strconv.Atoi(tickMSStr)
	if err != nil {
		return err
	}
	if tickMS < 0 {
		return strconv.ErrRange
	}

	tickInterval := time.Duration(tickMS) * time.Millisecond
	log.Printf("[streamer] starting replay: recording=%s, agent=%s, tick_interval=%v", recordingPath, agentURL, tickInterval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errReporter := reporter.NewLogReporter(os.Stderr)
	eventSource := source.NewTarGzEventSource(recordingPath, errReporter)
	agentClient, err := client.NewHTTPClient(agentURL, 30*time.Second)
	if err != nil {
		return err
	}

	realTimer := timer.NewRealTimer()
	replaySvc, err := service.NewReplayService(eventSource, agentClient, realTimer, tickInterval)
	if err != nil {
		return err
	}

	meta, count, err := replaySvc.Run(ctx)
	if err != nil {
		if ctx.Err() != nil {
			log.Printf("[streamer] replay cancelled after %d ticks", count)
			return nil
		}
		return err
	}

	log.Printf("[streamer] replay finished successfully for employee %s (session %s): dispatched %d ticks",
		meta.EmployeeID, meta.SessionID, count)
	return nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}
