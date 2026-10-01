// Package main is the composition root for the Central Audit Server service.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"assessment/modules/server/internal/adapters/handler"
	"assessment/modules/server/internal/adapters/repository"
	"assessment/modules/server/internal/service"
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
func Run(ctx context.Context, cfg Config) error {
	repo := repository.NewMemoryRepository()

	svc, err := service.NewService(repo)
	if err != nil {
		return fmt.Errorf("initialize service: %w", err)
	}

	h, err := handler.NewHTTPHandler(svc)
	if err != nil {
		return fmt.Errorf("initialize handler: %w", err)
	}

	httpServer := &http.Server{
		Addr:              cfg.Port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Port, err)
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("[server] listening on %s", ln.Addr().String())
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-ctx.Done():
		log.Println("[server] shutting down server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		log.Println("[server] server stopped")
		return nil
	case err := <-serverErr:
		return err
	}
}

func main() {
	if err := runMain(); err != nil {
		log.Printf("[server] fatal error: %v", err)
		os.Exit(1)
	}
}

func runMain() error {
	cfg := LoadConfig()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return Run(ctx, cfg)
}
