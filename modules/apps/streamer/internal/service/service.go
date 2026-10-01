// Package service implements the core event replay orchestration logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"assessment/modules/apps/streamer/internal/ports"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/session"
)

// Sentinel errors for ReplayService configuration.
var (
	ErrNilEventSource = errors.New("event source is required")
	ErrNilAgentClient = errors.New("agent client is required")
	ErrNilTimer       = errors.New("timer is required")
)

// ReplayService orchestrates streaming tick batches from an EventSource to an AgentClient.
type ReplayService struct {
	source       ports.EventSource
	client       ports.AgentClient
	timer        ports.Timer
	tickInterval time.Duration
}

// NewReplayService constructs and validates a new ReplayService instance.
func NewReplayService(
	source ports.EventSource,
	client ports.AgentClient,
	timer ports.Timer,
	tickInterval time.Duration,
) (*ReplayService, error) {
	if source == nil {
		return nil, ErrNilEventSource
	}
	if client == nil {
		return nil, ErrNilAgentClient
	}
	if timer == nil {
		return nil, ErrNilTimer
	}
	return &ReplayService{
		source:       source,
		client:       client,
		timer:        timer,
		tickInterval: tickInterval,
	}, nil
}

// Run executes the replay pipeline until all ticks are dispatched or ctx is cancelled.
func (s *ReplayService) Run(ctx context.Context) (session.Metadata, int, error) {
	meta, err := s.source.LoadMetadata(ctx)
	if err != nil {
		return session.Metadata{}, 0, fmt.Errorf("load metadata: %w", err)
	}

	tickCh, errCh, err := s.source.StreamTicks(ctx)
	if err != nil {
		return meta, 0, fmt.Errorf("stream ticks: %w", err)
	}

	sentCount, runErr := s.consumeStream(ctx, tickCh, errCh)
	return meta, sentCount, runErr
}

func (s *ReplayService) consumeStream(
	ctx context.Context,
	tickCh <-chan events.TickBatch,
	errCh <-chan error,
) (int, error) {
	sentCount := 0
	for {
		select {
		case <-ctx.Done():
			return sentCount, ctx.Err()
		case err, ok := <-errCh:
			if ok && err != nil {
				return sentCount, fmt.Errorf("stream error: %w", err)
			}
		case tick, ok := <-tickCh:
			if !ok {
				return handleTickClosed(sentCount, errCh)
			}
			if err := s.dispatchTick(ctx, tick); err != nil {
				return sentCount, err
			}
			sentCount++
		}
	}
}

func handleTickClosed(sentCount int, errCh <-chan error) (int, error) {
	select {
	case err, ok := <-errCh:
		if ok && err != nil {
			return sentCount, fmt.Errorf("stream error: %w", err)
		}
	default:
	}
	return sentCount, nil
}

func (s *ReplayService) dispatchTick(ctx context.Context, tick events.TickBatch) error {
	if err := s.client.SendTick(ctx, tick); err != nil {
		return fmt.Errorf("send tick %d: %w", tick.TickIndex, err)
	}

	if s.tickInterval <= 0 {
		return nil
	}

	if err := s.timer.Wait(ctx, s.tickInterval); err != nil {
		return fmt.Errorf("wait tick interval: %w", err)
	}
	return nil
}
