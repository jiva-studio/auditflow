// Package main is the entrypoint for the employee agent service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"assessment/modules/apps/agent/internal/adapters/client"
	"assessment/modules/apps/agent/internal/adapters/handler"
	"assessment/modules/apps/agent/internal/adapters/normalizer"
	"assessment/modules/apps/agent/internal/adapters/rules"
	"assessment/modules/apps/agent/internal/service"
	"assessment/modules/libs/telemetry"
)

var exitHandler = func(format string, args ...any) {
	slog.Default().Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

type appConfig struct {
	port          string
	rulesPath     string
	serverURL     string
	employeeID    string
	serverTimeout time.Duration
}

func main() {
	logger := telemetry.NewLogger(telemetry.LoggerConfig{
		ServiceName: "agent",
		Level:       slog.LevelInfo,
		Format:      telemetry.FormatJSON,
	})
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		exitHandler("[agent] fatal error: %v", err)
	}
}

func run(loggers ...*slog.Logger) error {
	var logger *slog.Logger
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	} else {
		logger = slog.Default()
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	server, err := buildServer(cfg, logger)
	if err != nil {
		return err
	}

	return runServer(server, cfg, logger)
}

func loadConfig() (appConfig, error) {
	port := getEnv("PORT", "8081")
	rulesPath := getEnv("RULES_PATH", "/rules.json")
	serverURL := getEnv("SERVER_URL", "http://server:8080")
	employeeID := getEnv("EMPLOYEE_ID", "emp-1")
	timeoutMSStr := getEnv("SERVER_TIMEOUT_MS", "5000")

	timeoutMS, err := strconv.Atoi(timeoutMSStr)
	if err != nil || timeoutMS <= 0 {
		return appConfig{}, fmt.Errorf("invalid SERVER_TIMEOUT_MS %q: must be positive integer", timeoutMSStr)
	}

	return appConfig{
		port:          port,
		rulesPath:     rulesPath,
		serverURL:     serverURL,
		employeeID:    employeeID,
		serverTimeout: time.Duration(timeoutMS) * time.Millisecond,
	}, nil
}

func buildServer(cfg appConfig, logger *slog.Logger) (*http.Server, error) {
	rulesProvider, err := rules.NewFileRulesProvider(cfg.rulesPath)
	if err != nil {
		return nil, fmt.Errorf("initialize rules provider: %w", err)
	}

	auditClient, err := client.NewAuditHTTPClient(cfg.serverURL, cfg.serverTimeout)
	if err != nil {
		return nil, fmt.Errorf("initialize audit client: %w", err)
	}

	windowNormalizer := normalizer.NewWindowNormalizer()
	agentService, err := service.NewService(
		cfg.employeeID,
		nil,
		rulesProvider.GetRules(),
		auditClient,
		service.WithNormalizer(windowNormalizer),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize agent service: %w", err)
	}

	httpHandler, err := handler.NewHTTPHandler(agentService, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize HTTP handler: %w", err)
	}

	return &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           httpHandler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

func runServer(server *http.Server, cfg appConfig, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("agent HTTP server started",
			slog.String("port", cfg.port),
			slog.String("employee_id", cfg.employeeID),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down agent server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		logger.Info("agent server stopped")
		return nil
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}
