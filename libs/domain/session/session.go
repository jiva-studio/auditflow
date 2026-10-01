// Package session defines employee recording session metadata and machine configurations.
package session

import (
	"errors"
	"strings"
	"time"

	"assessment/libs/domain/display"
	"assessment/libs/domain/geometry"
)

// Domain errors for session metadata validation.
var (
	ErrInvalidTimeRange  = errors.New("ended_at cannot be before started_at")
	ErrEmptyEmployeeID   = errors.New("employee id cannot be empty")
	ErrEmptySessionID    = errors.New("session id cannot be empty")
	ErrNoDisplaysDefined = errors.New("at least one display must be configured")
)

// TimeRange represents an immutable time interval.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange creates and validates a TimeRange value object.
func NewTimeRange(start, end time.Time) (TimeRange, error) {
	if end.Before(start) {
		return TimeRange{}, ErrInvalidTimeRange
	}
	return TimeRange{Start: start, End: end}, nil
}

// Duration returns the total duration of the time range.
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// Contains checks if a given timestamp falls within [Start, End].
func (tr TimeRange) Contains(t time.Time) bool {
	return (t.Equal(tr.Start) || t.After(tr.Start)) && (t.Equal(tr.End) || t.Before(tr.End))
}

// TicksCount returns the total number of 1-second ticks in this time range (rounded up).
func (tr TimeRange) TicksCount() int {
	dur := tr.Duration()
	if dur <= 0 {
		return 0
	}
	return int(dur.Seconds()) + 1
}

// MachineInfo represents information about the employee's machine.
type MachineInfo struct {
	Hostname  string
	OSVersion string
}

// Metadata represents metadata describing an employee recording session.
type Metadata struct {
	SessionID  string
	EmployeeID string
	TimeRange  TimeRange
	Machine    MachineInfo
	Displays   []display.Display
}

// NewMetadata creates and validates a Metadata entity.
func NewMetadata(sessionID, employeeID string, timeRange TimeRange, machine MachineInfo, displays []display.Display) (Metadata, error) {
	if strings.TrimSpace(sessionID) == "" {
		return Metadata{}, ErrEmptySessionID
	}
	if strings.TrimSpace(employeeID) == "" {
		return Metadata{}, ErrEmptyEmployeeID
	}
	if len(displays) == 0 {
		return Metadata{}, ErrNoDisplaysDefined
	}
	return Metadata{
		SessionID:  sessionID,
		EmployeeID: employeeID,
		TimeRange:  timeRange,
		Machine:    machine,
		Displays:   displays,
	}, nil
}

// FindDisplayForPoint finds the display that covers the given global desktop coordinate.
func (m Metadata) FindDisplayForPoint(p geometry.Point) (display.Display, bool) {
	for _, d := range m.Displays {
		if d.Contains(p) {
			return d, true
		}
	}
	return display.Display{}, false
}
