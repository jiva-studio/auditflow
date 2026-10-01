// Package main is the composition root for the Central Audit Server service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"assessment/modules/apps/server/internal/adapters/handler"
	"assessment/modules/apps/server/internal/adapters/repository"
	"assessment/modules/apps/server/internal/service"
	"assessment/modules/libs/telemetry"
)

// Config holds environment configuration for the server service.
type Config struct {
	Port string
}

// LoadConfig reads configuration from environment variables with fallbacks.
func LoadConfig() Config {
	port := os.Getenv("PORT")
	if strings.TrimSpace(port) == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	return Config{
		Port: port,
	}
}

// Run starts the server service and handles graceful shutdown.
func Run(ctx context.Context, cfg Config, loggers ...*slog.Logger) error {
	logger := getLogger(loggers...)

	httpServer, err := buildHTTPServer(cfg, logger)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Port, err)
	}

	return serveWithGracefulShutdown(ctx, httpServer, ln, logger)
}

func getLogger(loggers ...*slog.Logger) *slog.Logger {
	if len(loggers) > 0 && loggers[0] != nil {
		return loggers[0]
	}
	return slog.Default()
}

func buildHTTPServer(cfg Config, logger *slog.Logger) (*http.Server, error) {
	repo := repository.NewMemoryRepository()
	svc, err := service.NewService(repo)
	if err != nil {
		return nil, fmt.Errorf("initialize service: %w", err)
	}

	h, err := handler.NewHTTPHandler(svc, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize handler: %w", err)
	}

	return &http.Server{
		Addr:              cfg.Port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

func serveWithGracefulShutdown(ctx context.Context, server *http.Server, ln net.Listener, logger *slog.Logger) error {
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("audit server listening", slog.String("addr", ln.Addr().String()))
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		logger.Info("server stopped")
		return nil
	case err := <-serverErr:
		return err
	}
}

func main() {
	logger := telemetry.NewLogger(telemetry.LoggerConfig{
		ServiceName: "server",
		Level:       slog.LevelInfo,
		Format:      telemetry.FormatJSON,
	})
	slog.SetDefault(logger)

	if err := runMain(logger); err != nil {
		logger.Error("server fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func runMain(logger *slog.Logger) error {
	cfg := LoadConfig()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return Run(ctx, cfg, logger)
}
