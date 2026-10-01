// Package service implements the core application logic for the employee agent.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"assessment/modules/apps/agent/internal/ports"
	"assessment/modules/libs/domain/audit"
	"assessment/modules/libs/domain/desktop"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/rules"
)

// Sentinel errors for Service.
var (
	ErrEmptyEmployeeID   = errors.New("employee ID cannot be empty")
	ErrNilAuditClient    = errors.New("audit client cannot be nil")
	ErrDispatchPopupFail = errors.New("failed to dispatch audit popup")
	ErrNoRulesLoaded     = errors.New("no rules loaded")
)

// Option configures optional parameters for Service.
type Option func(*Service)

// WithNormalizer sets a custom ContextNormalizer strategy for the service.
func WithNormalizer(normalizer ports.ContextNormalizer) Option {
	return func(s *Service) {
		if normalizer != nil {
			s.normalizer = normalizer
		}
	}
}

type passthroughNormalizer struct{}

func (passthroughNormalizer) NormalizeClick(ctx rules.RuleEvaluationContext, m events.MouseEvent) rules.RuleEvaluationContext {
	if m.ProcessName != "" {
		ctx.Process = m.ProcessName
	}
	if m.WindowTitle != "" {
		ctx.WindowTitle = m.WindowTitle
	}
	return ctx
}

// Service coordinates desktop state updates, spatial hit-testing, rule evaluation, and audit popup emission.
type Service struct {
	employeeID        string
	state             *desktop.State
	rules             []rules.Rule
	auditClient       ports.AuditClient
	normalizer        ports.ContextNormalizer
	lastEmittedStates map[string]string
	mu                sync.Mutex
}

// NewService constructs and validates a new agent Service instance.
func NewService(employeeID string, state *desktop.State, rls []rules.Rule, auditClient ports.AuditClient, opts ...Option) (*Service, error) {
	if strings.TrimSpace(employeeID) == "" {
		return nil, ErrEmptyEmployeeID
	}
	if auditClient == nil {
		return nil, ErrNilAuditClient
	}
	if state == nil {
		state = desktop.NewState()
	}

	copiedRules := make([]rules.Rule, len(rls))
	copy(copiedRules, rls)

	s := &Service{
		employeeID:        employeeID,
		state:             state,
		rules:             copiedRules,
		auditClient:       auditClient,
		normalizer:        passthroughNormalizer{},
		lastEmittedStates: make(map[string]string),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s, nil
}

var _ ports.AgentService = (*Service)(nil)

// CheckReadiness validates that rules are loaded and ready.
func (s *Service) CheckReadiness(_ context.Context) error {
	s.mu.Lock()
	rulesCount := len(s.rules)
	s.mu.Unlock()

	if rulesCount == 0 {
		return ErrNoRulesLoaded
	}
	return nil
}

// ProcessTick processes an incoming batch of events in strict sequence.
func (s *Service) ProcessTick(ctx context.Context, batch events.TickBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ev := range batch.Events() {
		s.handleContextTransition(ev)
		s.state.Apply(ev)
		if err := s.processEvent(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) handleContextTransition(ev events.Event) {
	switch e := ev.(type) {
	case events.WindowEvent:
		if e.ProcessName != s.state.ActiveProcess() || e.WindowTitle != s.state.ActiveWindowTitle() {
			s.invalidateWindowRules()
		}
	case events.ClipboardEvent:
		if e.Text != s.state.ActiveClipboardText() {
			s.invalidateClipboardRules()
		}
	}
}

func (s *Service) invalidateWindowRules() {
	for _, r := range s.rules {
		if r.DependsOnWindow() && !r.DependsOnClipboard() {
			delete(s.lastEmittedStates, r.ID)
		}
	}
}

func (s *Service) invalidateClipboardRules() {
	for _, r := range s.rules {
		if r.DependsOnClipboard() {
			delete(s.lastEmittedStates, r.ID)
		}
	}
}

func (s *Service) processEvent(ctx context.Context, ev events.Event) error {
	switch e := ev.(type) {
	case events.MouseEvent:
		return s.handleMouseEvent(ctx, e)
	case events.ClipboardEvent:
		return s.handleStateChangeEvent(ctx, e.Timestamp)
	case events.WindowEvent:
		return s.handleStateChangeEvent(ctx, e.Timestamp)
	default:
		return nil
	}
}

func (s *Service) handleMouseEvent(ctx context.Context, m events.MouseEvent) error {
	if !m.IsClick {
		return nil
	}

	clickText, _ := s.state.FindTextAt(m.Position)
	evalCtx := s.state.BuildEvaluationContext(clickText)
	evalCtx = s.normalizer.NormalizeClick(evalCtx, m)
	return s.evaluateRulesAndDispatch(ctx, evalCtx, m.Timestamp, true)
}

func (s *Service) handleStateChangeEvent(ctx context.Context, ts time.Time) error {
	evalCtx := s.state.BuildEvaluationContext("")
	return s.evaluateRulesAndDispatch(ctx, evalCtx, ts, false)
}

func (s *Service) evaluateRulesAndDispatch(ctx context.Context, evalCtx rules.RuleEvaluationContext, ts time.Time, isClick bool) error {
	for _, r := range s.rules {
		if r.RequiresClick() != isClick {
			continue
		}

		title, body, matched := r.Evaluate(evalCtx)
		if !matched {
			continue
		}

		stateKey := fmt.Sprintf("%s|%s|%s", r.ID, title, body)
		if !isClick && s.lastEmittedStates[r.ID] == stateKey {
			continue
		}

		popup, err := audit.NewPopupWithTime(s.employeeID, r.ID, ts, title, body)
		if err != nil {
			continue
		}

		if err := s.auditClient.SendPopup(ctx, popup); err != nil {
			return fmt.Errorf("%w: %w", ErrDispatchPopupFail, err)
		}
		if !isClick {
			s.lastEmittedStates[r.ID] = stateKey
		}
	}
	return nil
}
