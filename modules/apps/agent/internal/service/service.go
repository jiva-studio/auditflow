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

// Service coordinates desktop state updates, spatial hit-testing, rule evaluation, and audit popup emission.
type Service struct {
	employeeID        string
	state             *desktop.State
	rules             []rules.Rule
	auditClient       ports.AuditClient
	lastEmittedStates map[string]string
	mu                sync.Mutex
}

// NewService constructs and validates a new agent Service instance.
func NewService(employeeID string, state *desktop.State, rls []rules.Rule, auditClient ports.AuditClient) (*Service, error) {
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

	return &Service{
		employeeID:        employeeID,
		state:             state,
		rules:             copiedRules,
		auditClient:       auditClient,
		lastEmittedStates: make(map[string]string),
	}, nil
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
		s.state.Apply(ev)
		if err := s.processEvent(ctx, ev); err != nil {
			return err
		}
	}
	return nil
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
	if !isClickEvent(m) {
		return nil
	}

	clickText, _ := s.state.FindTextAt(m.Position)
	evalCtx := s.state.BuildEvaluationContext(clickText)
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

func isGestureAction(act string) bool {
	switch act {
	case "move", "mousemove", "drag", "mousedrag", "scroll":
		return true
	default:
		return false
	}
}

func isPrimaryButton(btn string) bool {
	switch btn {
	case "left", "primary", "main":
		return true
	default:
		return false
	}
}

func isClickAction(act string) bool {
	switch act {
	case "click", "mousedown", "mouse_click":
		return true
	default:
		return false
	}
}

func hasClickCount(clickCount string) bool {
	if clickCount == "" {
		return false
	}
	cc := strings.ToLower(strings.TrimSpace(clickCount))
	return cc != "" && cc != "none" && cc != "0"
}

func isClickEvent(m events.MouseEvent) bool {
	act := strings.ToLower(strings.TrimSpace(m.Action))
	if isGestureAction(act) {
		return false
	}

	btn := strings.ToLower(strings.TrimSpace(m.Button))
	if btn == "middle" || btn == "right" {
		return false
	}

	return isClickAction(act) || isPrimaryButton(btn) || hasClickCount(m.ClickCount)
}
