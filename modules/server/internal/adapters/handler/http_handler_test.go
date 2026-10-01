package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"assessment/libs/domain/audit"
	v1 "assessment/libs/protocol/gen/go/v1"
	"assessment/modules/server/internal/adapters/handler"
	"assessment/modules/server/internal/adapters/mapper"
)

type mockServerService struct {
	recorded    []audit.Popup
	recordErr   error
	queryPopups []audit.Popup
	queryErr    error
	lastFilter  audit.Filter
}

func (m *mockServerService) RecordPopup(_ context.Context, popup audit.Popup) error {
	if m.recordErr != nil {
		return m.recordErr
	}
	m.recorded = append(m.recorded, popup)
	return nil
}

func (m *mockServerService) QueryPopups(_ context.Context, filter audit.Filter) ([]audit.Popup, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	m.lastFilter = filter
	return m.queryPopups, nil
}

func TestNewHTTPHandler_Validation(t *testing.T) {
	_, err := handler.NewHTTPHandler(nil)
	if !errors.Is(err, handler.ErrNilServerService) {
		t.Fatalf("expected ErrNilServerService, got %v", err)
	}

	svc := &mockServerService{}
	h, err := handler.NewHTTPHandler(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHTTPHandler_Health(t *testing.T) {
	svc := &mockServerService{}
	h, _ := handler.NewHTTPHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !stringsContains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	// Invalid method on /health
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/health", nil)
	h.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 on POST /health, got %d", recPost.Code)
	}
}

func TestHTTPHandler_RecordPopup_Protobuf(t *testing.T) {
	svc := &mockServerService{}
	h, _ := handler.NewHTTPHandler(svc)

	pb := &v1.Popup{
		Employee: "emp-1",
		Rule:     "forwarded-email-opened",
		Ts:       timestamppb.New(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)),
		Title:    "Forwarded email",
		Body:     "Body",
	}
	pbBytes, _ := proto.Marshal(pb)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(pbBytes))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d, body: %s", rec.Code, rec.Body.String())
	}
	if len(svc.recorded) != 1 || svc.recorded[0].Employee != "emp-1" {
		t.Fatalf("popup was not recorded properly")
	}
}

func TestHTTPHandler_RecordPopup_JSON(t *testing.T) {
	svc := &mockServerService{}
	h, _ := handler.NewHTTPHandler(svc)

	dto := mapper.PopupDTO{
		Employee: "emp-2",
		Rule:     "rule-2",
		TS:       "2026-03-10T12:00:00Z",
		Title:    "Title",
		Body:     "Body",
	}
	jsonBytes, _ := json.Marshal(dto)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d, body: %s", rec.Code, rec.Body.String())
	}
	if len(svc.recorded) != 1 || svc.recorded[0].Employee != "emp-2" {
		t.Fatalf("popup was not recorded properly")
	}
}

func TestHTTPHandler_RecordPopup_Errors(t *testing.T) {
	svc := &mockServerService{}
	h, _ := handler.NewHTTPHandler(svc)

	// 1. Invalid Protobuf
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader([]byte("not a proto")))
	h.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bad proto, got %d", rec1.Code)
	}

	// 2. Invalid JSON
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader([]byte("{invalid-json")))
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bad json, got %d", rec2.Code)
	}

	// 3. Validation failure in domain
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader([]byte(`{"employee":""}`)))
	req3.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on domain validation failure, got %d", rec3.Code)
	}

	// 4. Internal server error
	svc.recordErr = errors.New("database failure")
	dto := mapper.PopupDTO{
		Employee: "emp-2",
		Rule:     "rule-2",
		TS:       "2026-03-10T12:00:00Z",
		Title:    "Title",
		Body:     "Body",
	}
	validBytes, _ := json.Marshal(dto)
	rec4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(validBytes))
	req4.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on internal service error, got %d", rec4.Code)
	}

	// 5. Method not allowed on /audit
	rec5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodDelete, "/audit", nil)
	h.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 on DELETE /audit, got %d", rec5.Code)
	}
}

func TestHTTPHandler_RecordPopup_MaxBytesProtection(t *testing.T) {
	svc := &mockServerService{}
	h, _ := handler.NewHTTPHandler(svc)

	hugePopup := mapper.PopupDTO{
		Employee: "emp-1",
		Rule:     "rule-1",
		TS:       "2026-03-10T12:00:00Z",
		Title:    "Title",
		Body:     string(bytes.Repeat([]byte("x"), 5*1024*1024)),
	}
	hugeBytes, _ := json.Marshal(hugePopup)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(hugeBytes))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on oversized request body, got %d", rec.Code)
	}
}

func TestHTTPHandler_QueryPopups_JSON(t *testing.T) {
	p1, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title 1", "Body 1")
	svc := &mockServerService{
		queryPopups: []audit.Popup{p1},
	}
	h, _ := handler.NewHTTPHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/audit?employee=emp-1&rule=rule-1", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	if svc.lastFilter.Employee != "emp-1" || svc.lastFilter.Rule != "rule-1" {
		t.Fatalf("query parameters not mapped to filter properly: %+v", svc.lastFilter)
	}

	var popups []mapper.PopupDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &popups); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if len(popups) != 1 || popups[0].Title != "Title 1" {
		t.Fatalf("unexpected response payload: %+v", popups)
	}
}

func TestHTTPHandler_QueryPopups_Protobuf(t *testing.T) {
	p1, _ := audit.NewPopup("emp-1", "rule-1", "2026-03-10T12:00:00Z", "Title 1", "Body 1")
	svc := &mockServerService{
		queryPopups: []audit.Popup{p1},
	}
	h, _ := handler.NewHTTPHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	req.Header.Set("Accept", "application/x-protobuf")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var pbResp v1.AuditResponse
	if err := proto.Unmarshal(rec.Body.Bytes(), &pbResp); err != nil {
		t.Fatalf("failed to decode Protobuf response: %v", err)
	}
	if len(pbResp.GetPopups()) != 1 || pbResp.GetPopups()[0].GetTitle() != "Title 1" {
		t.Fatalf("unexpected protobuf payload: %+v", &pbResp)
	}
}

func TestHTTPHandler_QueryPopups_ServiceError(t *testing.T) {
	svc := &mockServerService{
		queryErr: errors.New("query failure"),
	}
	h, _ := handler.NewHTTPHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on query error, got %d", rec.Code)
	}
}

func stringsContains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
