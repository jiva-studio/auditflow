// Package handler provides HTTP transport adapters for the central audit server.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"google.golang.org/protobuf/proto"

	"assessment/modules/libs/domain/audit"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
	"assessment/modules/apps/server/internal/adapters/mapper"
	"assessment/modules/apps/server/internal/ports"
)

const maxPopupPayloadSize = 4 * 1024 * 1024 // 4 MB

// Sentinel errors for HTTPHandler.
var (
	ErrNilServerService = errors.New("server service is required")
)

// HTTPHandler exposes REST and Protobuf endpoints for the audit server.
type HTTPHandler struct {
	service ports.ServerService
	mux     *http.ServeMux
}

// NewHTTPHandler constructs and configures a new HTTPHandler.
func NewHTTPHandler(svc ports.ServerService) (*HTTPHandler, error) {
	if svc == nil {
		return nil, ErrNilServerService
	}

	h := &HTTPHandler{
		service: svc,
		mux:     http.NewServeMux(),
	}
	h.registerRoutes()
	return h, nil
}

// ServeHTTP implements http.Handler.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *HTTPHandler) registerRoutes() {
	h.mux.HandleFunc("/audit", h.handleAudit)
	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.HandleFunc("/health/live", h.handleLive)
	h.mux.HandleFunc("/health/ready", h.handleReady)
}

func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *HTTPHandler) handleLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"live"}`))
}

func (h *HTTPHandler) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func (h *HTTPHandler) handleAudit(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleRecordPopup(w, r)
	case http.MethodGet:
		h.handleQueryPopups(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPHandler) handleRecordPopup(w http.ResponseWriter, r *http.Request) {
	bodyReader := http.MaxBytesReader(w, r.Body, maxPopupPayloadSize)
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	popup, err := parseIncomingPopup(r.Header.Get("Content-Type"), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.RecordPopup(r.Context(), popup); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"status":"recorded"}`))
}

func parseIncomingPopup(contentType string, body []byte) (audit.Popup, error) {
	if strings.Contains(contentType, "application/json") {
		return parseJSONPopup(body)
	}
	return parseProtoPopup(body)
}

func parseJSONPopup(body []byte) (audit.Popup, error) {
	var dto mapper.PopupDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		return audit.Popup{}, errors.New("invalid JSON popup body")
	}
	return audit.NewPopup(dto.Employee, dto.Rule, dto.TS, dto.Title, dto.Body)
}

func parseProtoPopup(body []byte) (audit.Popup, error) {
	var pb v1.Popup
	if err := proto.Unmarshal(body, &pb); err != nil {
		return audit.Popup{}, errors.New("invalid Protobuf popup body")
	}
	return mapper.ToDomainPopup(&pb)
}

func (h *HTTPHandler) handleQueryPopups(w http.ResponseWriter, r *http.Request) {
	filter := audit.Filter{
		Employee: r.URL.Query().Get("employee"),
		Rule:     r.URL.Query().Get("rule"),
	}

	popups, err := h.service.QueryPopups(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.Contains(r.Header.Get("Accept"), "protobuf") {
		h.writeProtoResponse(w, popups)
		return
	}
	h.writeJSONResponse(w, popups)
}

func (h *HTTPHandler) writeProtoResponse(w http.ResponseWriter, popups []audit.Popup) {
	pbResp, err := mapper.ToProtoAuditResponse(popups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bytes, err := proto.Marshal(pbResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}

func (h *HTTPHandler) writeJSONResponse(w http.ResponseWriter, popups []audit.Popup) {
	respDTO := mapper.ToDTOList(popups)
	bytes, err := json.Marshal(respDTO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}
