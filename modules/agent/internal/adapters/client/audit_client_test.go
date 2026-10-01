package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"assessment/libs/domain/audit"
	v1 "assessment/libs/protocol/gen/go/v1"
)

func TestNewAuditHTTPClient(t *testing.T) {
	t.Run("empty server url", func(t *testing.T) {
		_, err := NewAuditHTTPClient("   ", 5*time.Second)
		if !errors.Is(err, ErrEmptyServerURL) {
			t.Errorf("expected ErrEmptyServerURL, got %v", err)
		}
	})

	t.Run("valid url", func(t *testing.T) {
		cli, err := NewAuditHTTPClient("http://localhost:8080/", 5*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cli.serverURL != "http://localhost:8080" {
			t.Errorf("expected trimmed server URL, got %s", cli.serverURL)
		}
	})
}

func TestAuditHTTPClient_SendPopup_Success(t *testing.T) {
	var receivedPB v1.Popup
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audit" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := proto.Unmarshal(bodyBytes, &receivedPB); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cli, err := NewAuditHTTPClient(srv.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected client creation error: %v", err)
	}

	p, err := audit.NewPopup("emp-1", "rule-123", time.Now().UTC().Format(time.RFC3339Nano), "Title", "Body")
	if err != nil {
		t.Fatalf("unexpected popup error: %v", err)
	}

	err = cli.SendPopup(context.Background(), p)
	if err != nil {
		t.Fatalf("SendPopup failed: %v", err)
	}

	if receivedPB.GetEmployee() != "emp-1" || receivedPB.GetRule() != "rule-123" {
		t.Errorf("unexpected employee %s or rule %s", receivedPB.GetEmployee(), receivedPB.GetRule())
	}
}

func TestAuditHTTPClient_SendPopup_Errors(t *testing.T) {
	t.Run("server error status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		cli, _ := NewAuditHTTPClient(srv.URL, 5*time.Second)
		p, _ := audit.NewPopup("emp-1", "rule-1", time.Now().UTC().Format(time.RFC3339Nano), "T", "B")
		err := cli.SendPopup(context.Background(), p)
		if !errors.Is(err, ErrNonSuccessStatus) {
			t.Errorf("expected ErrNonSuccessStatus, got %v", err)
		}
	})

	t.Run("invalid request context / connection refused", func(t *testing.T) {
		cli, _ := NewAuditHTTPClient("http://127.0.0.1:59999", 50*time.Millisecond)
		p, _ := audit.NewPopup("emp-1", "rule-1", time.Now().UTC().Format(time.RFC3339Nano), "T", "B")
		err := cli.SendPopup(context.Background(), p)
		if err == nil {
			t.Error("expected connection error")
		}
	})
}
