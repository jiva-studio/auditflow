// Package source implements the event source adapter for reading tar.gz session recordings.
package source

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"assessment/libs/domain/display"
	"assessment/libs/domain/events"
	"assessment/libs/domain/geometry"
	"assessment/libs/domain/session"
	"assessment/modules/streamer/internal/adapters/reporter"
	"assessment/modules/streamer/internal/ports"
)

// Sentinel errors for TarGzEventSource.
var (
	ErrRecordingNotFound        = errors.New("recording archive not found")
	ErrCorruptArchive           = errors.New("corrupt recording archive")
	ErrMetadataNotFound         = errors.New("metadata.json not found in archive")
	ErrUnsupportedSchemaVersion = errors.New("unsupported metadata schema version")
)

// TarGzEventSource reads session recordings from a .tar.gz archive.
type TarGzEventSource struct {
	archivePath string
	reporter    ports.ErrorReporter
}

// NewTarGzEventSource creates a new TarGzEventSource with the given reporter (defaults to stderr LogReporter if nil).
func NewTarGzEventSource(archivePath string, rep ports.ErrorReporter) *TarGzEventSource {
	if rep == nil {
		rep = reporter.NewLogReporter(nil)
	}
	return &TarGzEventSource{
		archivePath: archivePath,
		reporter:    rep,
	}
}

var _ ports.EventSource = (*TarGzEventSource)(nil)

type rawMetadata struct {
	SchemaVersion string     `json:"schema_version"`
	SessionID     string     `json:"session_id"`
	EmployeeID    string     `json:"employee_id"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       time.Time  `json:"ended_at"`
	Machine       rawMachine `json:"machine"`
}

type rawMachine struct {
	Hostname  string       `json:"hostname"`
	OSVersion string       `json:"os_version"`
	Displays  []rawDisplay `json:"displays"`
}

type rawDisplay struct {
	ID      int     `json:"id"`
	Bounds  [4]int  `json:"bounds"`
	Scale   float64 `json:"scale"`
	Primary bool    `json:"primary"`
}

// LoadMetadata extracts and parses metadata.json from the tar.gz archive.
func (s *TarGzEventSource) LoadMetadata(ctx context.Context) (session.Metadata, error) {
	f, err := os.Open(s.archivePath)
	if err != nil {
		return session.Metadata{}, fmt.Errorf("%w: %s", ErrRecordingNotFound, err.Error())
	}
	defer func() {
		_ = f.Close()
	}()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return session.Metadata{}, fmt.Errorf("%w: open gzip reader: %s", ErrCorruptArchive, err.Error())
	}
	defer func() {
		_ = gzr.Close()
	}()

	tr := tar.NewReader(gzr)
	for {
		if err := ctx.Err(); err != nil {
			return session.Metadata{}, err
		}

		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return session.Metadata{}, fmt.Errorf("%w: read tar entry: %s", ErrCorruptArchive, err.Error())
		}

		if strings.HasSuffix(header.Name, "metadata.json") {
			return s.decodeMetadataHeader(tr)
		}
	}

	return session.Metadata{}, ErrMetadataNotFound
}

func (s *TarGzEventSource) decodeMetadataHeader(r io.Reader) (session.Metadata, error) {
	var raw rawMetadata
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return session.Metadata{}, fmt.Errorf("%w: decode metadata: %s", ErrCorruptArchive, err.Error())
	}
	if raw.SchemaVersion != "" && !strings.HasPrefix(raw.SchemaVersion, "1.") {
		s.reporter.ReportWarning("non-standard schema version detected", map[string]any{
			"schema_version": raw.SchemaVersion,
		})
	}
	return mapRawMetadata(raw)
}

func mapRawMetadata(raw rawMetadata) (session.Metadata, error) {
	timeRange, err := session.NewTimeRange(raw.StartedAt, raw.EndedAt)
	if err != nil {
		return session.Metadata{}, fmt.Errorf("invalid time range: %w", err)
	}

	var displays []display.Display
	for _, rd := range raw.Machine.Displays {
		bounds, err := geometry.NewRectangle(rd.Bounds[0], rd.Bounds[1], rd.Bounds[2], rd.Bounds[3])
		if err != nil {
			return session.Metadata{}, fmt.Errorf("invalid display bounds: %w", err)
		}
		d, err := display.NewDisplay(rd.ID, bounds, rd.Scale, rd.Primary)
		if err != nil {
			return session.Metadata{}, fmt.Errorf("invalid display: %w", err)
		}
		displays = append(displays, d)
	}

	machineInfo := session.MachineInfo{
		Hostname:  raw.Machine.Hostname,
		OSVersion: raw.Machine.OSVersion,
	}

	return session.NewMetadata(
		raw.SessionID,
		raw.EmployeeID,
		timeRange,
		machineInfo,
		displays,
	)
}

// StreamTicks reads all events, organizes them into 1-second interval TickBatches, and streams them.
func (s *TarGzEventSource) StreamTicks(ctx context.Context) (<-chan events.TickBatch, <-chan error, error) {
	meta, err := s.LoadMetadata(ctx)
	if err != nil {
		return nil, nil, err
	}

	allEvents, err := s.readAllEvents(ctx)
	if err != nil {
		return nil, nil, err
	}

	sort.SliceStable(allEvents, func(i, j int) bool {
		return allEvents[i].GetTimestamp().Before(allEvents[j].GetTimestamp())
	})

	batches, err := partitionIntoTicks(meta.TimeRange.Start, meta.TimeRange.End, allEvents)
	if err != nil {
		return nil, nil, err
	}

	tickCh := make(chan events.TickBatch)
	errCh := make(chan error, 1)

	go func() {
		defer close(tickCh)
		defer close(errCh)

		for _, batch := range batches {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case tickCh <- batch:
			}
		}
	}()

	return tickCh, errCh, nil
}

func (s *TarGzEventSource) readAllEvents(ctx context.Context) ([]events.Event, error) {
	f, err := os.Open(s.archivePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRecordingNotFound, err.Error())
	}
	defer func() {
		_ = f.Close()
	}()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("%w: open gzip reader: %s", ErrCorruptArchive, err.Error())
	}
	defer func() {
		_ = gzr.Close()
	}()

	var allEvents []events.Event
	tr := tar.NewReader(gzr)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: read tar entry: %s", ErrCorruptArchive, err.Error())
		}

		baseName := filepath.Base(header.Name)
		evs, err := s.parseEventFile(baseName, tr)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", header.Name, err)
		}
		allEvents = append(allEvents, evs...)
	}

	return allEvents, nil
}

func (s *TarGzEventSource) parseEventFile(filename string, r io.Reader) ([]events.Event, error) {
	scanner := bufio.NewScanner(r)
	const maxScanCapacity = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScanCapacity)

	var result []events.Event
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		ev, err := parseEventLine(filename, line)
		if err != nil {
			s.reporter.ReportWarning("malformed event line skipped", map[string]any{
				"filename": filename,
				"line":     lineNum,
				"error":    err.Error(),
			})
			continue
		}
		if ev != nil {
			result = append(result, ev)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: scanner error in %s: %s", ErrCorruptArchive, filename, err.Error())
	}

	return result, nil
}

func parseEventLine(filename string, line []byte) (events.Event, error) {
	switch filename {
	case "windows.jsonl":
		return parseWindowEvent(line)
	case "mouse.jsonl":
		return parseMouseEvent(line)
	case "clipboard.jsonl":
		return parseClipboardEvent(line)
	case "ocr.jsonl":
		return parseOCREvent(line)
	case "keyboard.jsonl", "keystrokes.jsonl", "mouse_scroll.jsonl", "mouse_drag.jsonl":
		return parseGenericEvent(filename, line)
	default:
		return nil, nil
	}
}

type rawWindowEvent struct {
	TS          time.Time `json:"ts"`
	Event       string    `json:"event"`
	WindowTitle string    `json:"window_title"`
	ProcessName string    `json:"process_name"`
	AppName     string    `json:"application_name"`
	WindowRect  [4]int    `json:"window_rect"`
	WindowState string    `json:"window_state"`
	URL         string    `json:"url"`
	Domain      string    `json:"domain"`
	DwellTimeMS int       `json:"dwell_time_ms"`
}

func parseWindowEvent(data []byte) (events.Event, error) {
	var r rawWindowEvent
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal window event: %w", err)
	}
	rect, _ := geometry.NewRectangle(r.WindowRect[0], r.WindowRect[1], r.WindowRect[2], r.WindowRect[3])
	return events.WindowEvent{
		Timestamp:       r.TS,
		Action:          r.Event,
		WindowTitle:     r.WindowTitle,
		ProcessName:     r.ProcessName,
		ApplicationName: r.AppName,
		WindowRect:      rect,
		WindowState:     r.WindowState,
		URL:             r.URL,
		Domain:          r.Domain,
		DwellTimeMS:     r.DwellTimeMS,
	}, nil
}

type rawMouseEvent struct {
	TS          time.Time `json:"ts"`
	Event       string    `json:"event"`
	Button      string    `json:"button"`
	MouseX      int       `json:"mouse_x"`
	MouseY      int       `json:"mouse_y"`
	ClickCount  string    `json:"click_count"`
	WindowTitle string    `json:"window_title"`
	ProcessName string    `json:"process_name"`
}

func parseMouseEvent(data []byte) (events.Event, error) {
	var r rawMouseEvent
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal mouse event: %w", err)
	}
	pos := geometry.Point{X: r.MouseX, Y: r.MouseY}
	return events.MouseEvent{
		Timestamp:   r.TS,
		Action:      r.Event,
		Button:      r.Button,
		Position:    pos,
		ClickCount:  r.ClickCount,
		WindowTitle: r.WindowTitle,
		ProcessName: r.ProcessName,
	}, nil
}

type rawClipboardEvent struct {
	TS        time.Time `json:"ts"`
	Event     string    `json:"event"`
	Text      string    `json:"clipboard_content_text"`
	Length    int       `json:"clipboard_content_length"`
	SourceApp string    `json:"clipboard_source_app"`
	Formats   []string  `json:"clipboard_format"`
}

func parseClipboardEvent(data []byte) (events.Event, error) {
	var r rawClipboardEvent
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal clipboard event: %w", err)
	}
	return events.ClipboardEvent{
		Timestamp: r.TS,
		Action:    r.Event,
		Text:      r.Text,
		Length:    r.Length,
		SourceApp: r.SourceApp,
		Formats:   r.Formats,
	}, nil
}

type rawOCREvent struct {
	TS               time.Time         `json:"ts"`
	Filename         string            `json:"filename"`
	DisplayID        int               `json:"display_id"`
	Resolution       [2]int            `json:"resolution"`
	ScaleFactor      float64           `json:"display_scale_factor"`
	WindowRect       [4]int            `json:"window_rect"`
	Blocks           []rawOCRTextBlock `json:"ocr_text_blocks"`
	DeduplicatedFrom string            `json:"deduplicated_from"`
}

type rawOCRTextBlock struct {
	Text        string  `json:"text"`
	BoundingBox [4]int  `json:"bounding_box"`
	Confidence  float64 `json:"confidence"`
}

func parseOCREvent(data []byte) (events.Event, error) {
	var r rawOCREvent
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal ocr event: %w", err)
	}
	res := geometry.Size{Width: r.Resolution[0], Height: r.Resolution[1]}
	rect, _ := geometry.NewRectangle(r.WindowRect[0], r.WindowRect[1], r.WindowRect[2], r.WindowRect[3])

	var blocks []display.OCRTextBlock
	for _, b := range r.Blocks {
		box, _ := geometry.NewRectangle(b.BoundingBox[0], b.BoundingBox[1], b.BoundingBox[2], b.BoundingBox[3])
		blocks = append(blocks, display.OCRTextBlock{
			Text:       b.Text,
			Box:        box,
			Confidence: b.Confidence,
		})
	}

	return events.OCREvent{
		Timestamp:        r.TS,
		Filename:         r.Filename,
		DisplayID:        r.DisplayID,
		Resolution:       res,
		ScaleFactor:      r.ScaleFactor,
		WindowRect:       rect,
		Blocks:           blocks,
		DeduplicatedFrom: r.DeduplicatedFrom,
	}, nil
}

func parseGenericEvent(filename string, data []byte) (events.Event, error) {
	var r struct {
		TS time.Time `json:"ts"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal generic event: %w", err)
	}
	var details map[string]any
	_ = json.Unmarshal(data, &details)

	evType := mapFilenameToEventType(filename)
	return events.GenericActivityEvent{
		Timestamp: r.TS,
		Type:      evType,
		Details:   details,
	}, nil
}

func mapFilenameToEventType(filename string) events.EventType {
	switch filename {
	case "keyboard.jsonl":
		return events.EventTypeKeyboard
	case "keystrokes.jsonl":
		return events.EventTypeKeystroke
	case "mouse_scroll.jsonl":
		return events.EventTypeMouseScroll
	case "mouse_drag.jsonl":
		return events.EventTypeMouseDrag
	default:
		return events.EventType("unknown")
	}
}

func partitionIntoTicks(startedAt, endedAt time.Time, sortedEvents []events.Event) ([]events.TickBatch, error) {
	if endedAt.Before(startedAt) {
		return nil, fmt.Errorf("ended_at %v is before started_at %v", endedAt, startedAt)
	}

	var batches []events.TickBatch
	currStart := startedAt
	eventIdx := 0
	tickIdx := 0

	for currStart.Before(endedAt) {
		currEnd := currStart.Add(time.Second)
		if currEnd.After(endedAt) {
			currEnd = endedAt
		}

		batch, err := events.NewTickBatch(tickIdx, currStart, currEnd)
		if err != nil {
			return nil, err
		}

		for eventIdx < len(sortedEvents) {
			ev := sortedEvents[eventIdx]
			ts := ev.GetTimestamp()

			if ts.Before(currStart) {
				eventIdx++
				continue
			}
			if !ts.Before(currEnd) {
				break
			}

			_ = batch.Add(ev)
			eventIdx++
		}

		batches = append(batches, batch)
		currStart = currEnd
		tickIdx++
	}

	return batches, nil
}
