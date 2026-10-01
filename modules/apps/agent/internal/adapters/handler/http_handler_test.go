package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/modules/libs/domain/events"
	v1 "assessment/modules/libs/protocol/gen/go/v1"
)

type mockService struct {
	processErr   error
	readinessErr error
	lastBatch    events.TickBatch
}

func (m *mockService) ProcessTick(_ context.Context, batch events.TickBatch) error {
	if m.processErr != nil {
		return m.processErr
	}
	m.lastBatch = batch
	return nil
}

func (m *mockService) CheckReadiness(_ context.Context) error {
	return m.readinessErr
}

func TestNewHTTPHandler(t *testing.T) {
	t.Run("nil service", func(t *testing.T) {
		_, err := NewHTTPHandler(nil)
		if !errors.Is(err, ErrNilAgentService) {
			t.Errorf("expected ErrNilAgentService, got %v", err)
		}
	})

	t.Run("valid service", func(t *testing.T) {
		h, err := NewHTTPHandler(&mockService{})
		if err != nil || h == nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestHTTPHandler_Health(t *testing.T) {
	h, _ := NewHTTPHandler(&mockService{})
	router := h.Routes()

	t.Run("GET /health success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("POST /health method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})
}

func TestHTTPHandler_LiveAndReady(t *testing.T) {
	mock := &mockService{}
	h, _ := NewHTTPHandler(mock)
	router := h.Routes()

	t.Run("GET /health/live success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("POST /health/live method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/health/live", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})

	t.Run("GET /health/ready success", func(t *testing.T) {
		mock.readinessErr = nil
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("GET /health/ready failure", func(t *testing.T) {
		mock.readinessErr = errors.New("upstream offline")
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("POST /health/ready method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/health/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})
}

func TestHTTPHandler_Tick_Success(t *testing.T) {
	svc := &mockService{}
	h, _ := NewHTTPHandler(svc)
	router := h.Routes()

	now := time.Now().UTC()
	pb := &v1.TickBatch{
		TickIndex: 42,
		StartTime: timestamppb.New(now),
		EndTime:   timestamppb.New(now.Add(time.Second)),
	}
	data, _ := proto.Marshal(pb)

	req := httptest.NewRequest(http.MethodPost, "/tick", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (%s)", w.Code, w.Body.String())
	}
	if svc.lastBatch.TickIndex != 42 {
		t.Errorf("expected batch index 42, got %d", svc.lastBatch.TickIndex)
	}
}

func TestHTTPHandler_Tick_Errors(t *testing.T) {
	svc := &mockService{}
	h, _ := NewHTTPHandler(svc)
	router := h.Routes()

	t.Run("GET /tick method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tick", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})

	t.Run("POST /tick invalid protobuf", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/tick", bytes.NewReader([]byte{0xFF, 0xFF, 0xFF}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("POST /tick service error", func(t *testing.T) {
		errSvc := &mockService{processErr: errors.New("processing failed")}
		errH, _ := NewHTTPHandler(errSvc)
		errRouter := errH.Routes()

		now := time.Now().UTC()
		pb := &v1.TickBatch{TickIndex: 1, StartTime: timestamppb.New(now), EndTime: timestamppb.New(now.Add(time.Second))}
		data, _ := proto.Marshal(pb)

		req := httptest.NewRequest(http.MethodPost, "/tick", bytes.NewReader(data))
		w := httptest.NewRecorder()
		errRouter.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
