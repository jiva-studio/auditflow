package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/timer"
	"assessment/modules/apps/streamer/internal/service"
	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/geometry"
	"assessment/modules/libs/domain/session"
)

type mockSource struct {
	meta      session.Metadata
	metaErr   error
	ticks     []events.TickBatch
	err       error
	streamErr error
}

func (m *mockSource) LoadMetadata(_ context.Context) (session.Metadata, error) {
	if m.metaErr != nil {
		return session.Metadata{}, m.metaErr
	}
	return m.meta, nil
}

func (m *mockSource) StreamTicks(_ context.Context) (<-chan events.TickBatch, <-chan error, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	tickCh := make(chan events.TickBatch, len(m.ticks))
	errCh := make(chan error, 1)

	for _, t := range m.ticks {
		tickCh <- t
	}
	if m.streamErr != nil {
		errCh <- m.streamErr
	}
	close(tickCh)
	close(errCh)

	return tickCh, errCh, nil
}

type mockClient struct {
	sent []events.TickBatch
	err  error
}

func (m *mockClient) SendTick(_ context.Context, batch events.TickBatch) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, batch)
	return nil
}

type errTimer struct{}

func (e errTimer) Wait(_ context.Context, _ time.Duration) error {
	return errors.New("timer failed")
}

func createTestMeta() session.Metadata {
	d, _ := display.NewDisplay(0, geometry.Rectangle{X: 0, Y: 0, Width: 1920, Height: 1080}, 1.0, true)
	tr, _ := session.NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	meta, _ := session.NewMetadata("sess-1", "emp-1", tr, session.MachineInfo{Hostname: "HOST-1"}, []display.Display{d})
	return meta
}

func TestReplayService_Validation(t *testing.T) {
	src := &mockSource{meta: createTestMeta()}
	cli := &mockClient{}
	tmr := timer.NewMockTimer()

	if _, err := service.NewReplayService(nil, cli, tmr, time.Millisecond); !errors.Is(err, service.ErrNilEventSource) {
		t.Fatalf("expected ErrNilEventSource, got %v", err)
	}
	if _, err := service.NewReplayService(src, nil, tmr, time.Millisecond); !errors.Is(err, service.ErrNilAgentClient) {
		t.Fatalf("expected ErrNilAgentClient, got %v", err)
	}
	if _, err := service.NewReplayService(src, cli, nil, time.Millisecond); !errors.Is(err, service.ErrNilTimer) {
		t.Fatalf("expected ErrNilTimer, got %v", err)
	}
}

func TestReplayService_SuccessfulRun(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	b1, _ := events.NewTickBatch(0, start, start.Add(time.Second))
	b2, _ := events.NewTickBatch(1, start.Add(time.Second), start.Add(2*time.Second))

	src := &mockSource{
		meta:  createTestMeta(),
		ticks: []events.TickBatch{b1, b2},
	}
	cli := &mockClient{}
	tmr := timer.NewMockTimer()

	svc, err := service.NewReplayService(src, cli, tmr, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	meta, count, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 ticks sent, got %d", count)
	}
	if meta.EmployeeID != "emp-1" {
		t.Fatalf("expected emp-1 metadata, got %s", meta.EmployeeID)
	}
	if len(cli.sent) != 2 {
		t.Fatalf("expected 2 sent batches in client, got %d", len(cli.sent))
	}
	if len(tmr.WaitedDurations) != 2 {
		t.Fatalf("expected 2 timer waits, got %d", len(tmr.WaitedDurations))
	}
}

func TestReplayService_Errors(t *testing.T) {
	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	b1, _ := events.NewTickBatch(0, start, start.Add(time.Second))

	t.Run("load metadata error", func(t *testing.T) {
		src := &mockSource{metaErr: errors.New("load error")}
		svc, _ := service.NewReplayService(src, &mockClient{}, timer.NewMockTimer(), time.Millisecond)
		_, _, err := svc.Run(context.Background())
		if err == nil {
			t.Fatalf("expected load metadata error, got nil")
		}
	})

	t.Run("stream ticks init error", func(t *testing.T) {
		src := &mockSource{meta: createTestMeta(), err: errors.New("stream init error")}
		svc, _ := service.NewReplayService(src, &mockClient{}, timer.NewMockTimer(), time.Millisecond)
		_, _, err := svc.Run(context.Background())
		if err == nil {
			t.Fatalf("expected stream init error, got nil")
		}
	})

	t.Run("stream channel error", func(t *testing.T) {
		src := &mockSource{meta: createTestMeta(), streamErr: errors.New("stream runtime error")}
		svc, _ := service.NewReplayService(src, &mockClient{}, timer.NewMockTimer(), time.Millisecond)
		_, _, err := svc.Run(context.Background())
		if err == nil {
			t.Fatalf("expected stream channel error, got nil")
		}
	})

	t.Run("client send error", func(t *testing.T) {
		src := &mockSource{meta: createTestMeta(), ticks: []events.TickBatch{b1}}
		cli := &mockClient{err: errors.New("send error")}
		svc, _ := service.NewReplayService(src, cli, timer.NewMockTimer(), time.Millisecond)
		_, _, err := svc.Run(context.Background())
		if err == nil {
			t.Fatalf("expected send error, got nil")
		}
	})

	t.Run("timer wait error", func(t *testing.T) {
		src := &mockSource{meta: createTestMeta(), ticks: []events.TickBatch{b1}}
		svc, _ := service.NewReplayService(src, &mockClient{}, errTimer{}, time.Millisecond)
		_, _, err := svc.Run(context.Background())
		if err == nil {
			t.Fatalf("expected timer wait error, got nil")
		}
	})
}

func TestReplayService_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	b1, _ := events.NewTickBatch(0, start, start.Add(time.Second))

	src := &mockSource{
		meta:  createTestMeta(),
		ticks: []events.TickBatch{b1},
	}
	cli := &mockClient{}
	tmr := timer.NewMockTimer()

	svc, _ := service.NewReplayService(src, cli, tmr, 10*time.Millisecond)
	_, _, err := svc.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
