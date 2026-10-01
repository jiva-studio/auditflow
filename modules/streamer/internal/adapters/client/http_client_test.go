package client_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"assessment/libs/domain/events"
	"assessment/libs/domain/geometry"
	v1 "assessment/libs/protocol/gen/go/v1"
	"assessment/modules/streamer/internal/adapters/client"
)

func TestHTTPClient_SendTick_Success(t *testing.T) {
	var receivedBody []byte
	var receivedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tick" {
			t.Errorf("got path %s, want /tick", r.URL.Path)
		}
		receivedHeader = r.Header.Get("Content-Type")
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body error: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := client.NewHTTPClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Second)
	batch, _ := events.NewTickBatch(1, start, end)
	_ = batch.Add(events.MouseEvent{
		Timestamp: start.Add(100 * time.Millisecond),
		Action:    "click",
		Button:    "left",
		Position:  geometry.Point{X: 100, Y: 200},
	})

	if err := c.SendTick(context.Background(), batch); err != nil {
		t.Fatalf("unexpected SendTick error: %v", err)
	}

	if receivedHeader != "application/x-protobuf" {
		t.Errorf("got Content-Type %s, want application/x-protobuf", receivedHeader)
	}

	var pbBatch v1.TickBatch
	if err := proto.Unmarshal(receivedBody, &pbBatch); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if pbBatch.TickIndex != 1 {
		t.Errorf("got tick_index %d, want 1", pbBatch.TickIndex)
	}
	if len(pbBatch.Events) != 1 {
		t.Errorf("got %d events, want 1", len(pbBatch.Events))
	}
}

func TestHTTPClient_SendTick_Errors(t *testing.T) {
	t.Run("server error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c, _ := client.NewHTTPClient(server.URL, time.Second)
		start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
		end := start.Add(time.Second)
		batch, _ := events.NewTickBatch(0, start, end)

		err := c.SendTick(context.Background(), batch)
		if err == nil {
			t.Fatalf("expected error on 500 status, got nil")
		}
	})

	t.Run("empty agent url", func(t *testing.T) {
		_, err := client.NewHTTPClient("", time.Second)
		if err == nil {
			t.Fatalf("expected error on empty url, got nil")
		}
	})

	t.Run("dispatch network error", func(t *testing.T) {
		c, _ := client.NewHTTPClient("http://127.0.0.1:59999", 50*time.Millisecond)
		start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
		end := start.Add(time.Second)
		batch, _ := events.NewTickBatch(0, start, end)

		err := c.SendTick(context.Background(), batch)
		if err == nil {
			t.Fatalf("expected network error, got nil")
		}
	})
}
