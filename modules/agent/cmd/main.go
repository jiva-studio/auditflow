// Package main is the entrypoint for the employee agent service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"assessment/modules/agent/internal/adapters/client"
	"assessment/modules/agent/internal/adapters/handler"
	"assessment/modules/agent/internal/adapters/rules"
	"assessment/modules/agent/internal/service"
)

var exitHandler = log.Fatalf

type appConfig struct {
	port          string
	rulesPath     string
	serverURL     string
	employeeID    string
	serverTimeout time.Duration
}

func main() {
	if err := run(); err != nil {
		exitHandler("[agent] fatal error: %v", err)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	server, err := buildServer(cfg)
	if err != nil {
		return err
	}

	return runServer(server, cfg)
}

func loadConfig() (appConfig, error) {
	port := getEnv("PORT", "8081")
	rulesPath := getEnv("RULES_PATH", "/data/rules.json")
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

func buildServer(cfg appConfig) (*http.Server, error) {
	rulesProvider, err := rules.NewFileRulesProvider(cfg.rulesPath)
	if err != nil {
		return nil, fmt.Errorf("initialize rules provider: %w", err)
	}

	auditClient, err := client.NewAuditHTTPClient(cfg.serverURL, cfg.serverTimeout)
	if err != nil {
		return nil, fmt.Errorf("initialize audit client: %w", err)
	}

	agentService, err := service.NewService(cfg.employeeID, nil, rulesProvider.GetRules(), auditClient)
	if err != nil {
		return nil, fmt.Errorf("initialize agent service: %w", err)
	}

	httpHandler, err := handler.NewHTTPHandler(agentService)
	if err != nil {
		return nil, fmt.Errorf("initialize HTTP handler: %w", err)
	}

	return &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           httpHandler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

func runServer(server *http.Server, cfg appConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("[agent] listening on :%s for employee %s", cfg.port, cfg.employeeID)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Printf("[agent] shutting down server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		log.Printf("[agent] server stopped")
		return nil
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}
