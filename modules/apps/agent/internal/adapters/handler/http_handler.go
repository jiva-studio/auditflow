// Package handler implements the inbound HTTP transport adapter.
package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"google.golang.org/protobuf/proto"

	"assessment/modules/apps/agent/internal/adapters/mapper"
	"assessment/modules/apps/agent/internal/ports"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
	"assessment/modules/libs/telemetry"
)

const maxTickPayloadSize = 16 * 1024 * 1024 // 16 MB

// Sentinel errors for HTTPHandler.
var (
	ErrNilAgentService = errors.New("agent service cannot be nil")
)

// HTTPHandler serves inbound HTTP requests for the agent service.
type HTTPHandler struct {
	service ports.AgentService
	logger  *slog.Logger
}

// NewHTTPHandler creates a new HTTPHandler wrapping the given AgentService.
func NewHTTPHandler(service ports.AgentService, loggers ...*slog.Logger) (*HTTPHandler, error) {
	if service == nil {
		return nil, ErrNilAgentService
	}
	var logger *slog.Logger
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	} else {
		logger = slog.Default()
	}
	return &HTTPHandler{
		service: service,
		logger:  logger,
	}, nil
}

// Routes registers the HTTP endpoints and returns the root handler wrapped with TraceMiddleware.
func (h *HTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tick", h.handleTick)
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/health/live", h.handleLive)
	mux.HandleFunc("/health/ready", h.handleReady)
	return telemetry.TraceMiddleware("agent", h.logger)(mux)
}

func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, `{"status":"ok"}`)
}

func (h *HTTPHandler) handleLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, `{"status":"live"}`)
}

func (h *HTTPHandler) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := h.service.CheckReadiness(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, "{\"status\":\"not ready\",\"error\":%q}\n", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, `{"status":"ready"}`)
}

func (h *HTTPHandler) handleTick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer func() { _ = r.Body.Close() }()

	bodyReader := http.MaxBytesReader(w, r.Body, maxTickPayloadSize)
	data, err := io.ReadAll(bodyReader)
	if err != nil {
		http.Error(w, fmt.Sprintf("read request body: %v", err), http.StatusBadRequest)
		return
	}

	var pbBatch v1.TickBatch
	if err := proto.Unmarshal(data, &pbBatch); err != nil {
		http.Error(w, fmt.Sprintf("unmarshal protobuf: %v", err), http.StatusBadRequest)
		return
	}

	domainBatch := mapper.ToDomainTickBatch(&pbBatch)
	if err := h.service.ProcessTick(r.Context(), domainBatch); err != nil {
		http.Error(w, fmt.Sprintf("process tick: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, `{"status":"ok"}`)
}
