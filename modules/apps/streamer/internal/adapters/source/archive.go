// Package source implements the event source adapter for reading tar.gz session recordings.
package source

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"assessment/modules/apps/streamer/internal/adapters/reporter"
	"assessment/modules/apps/streamer/internal/ports"
	"assessment/modules/libs/domain/display"
	"assessment/modules/libs/domain/events"
	"assessment/modules/libs/domain/geometry"
	"assessment/modules/libs/domain/session"
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
	metaMu      sync.RWMutex
	cachedMeta  *session.Metadata
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

func (s *TarGzEventSource) getCachedMetadata() (session.Metadata, bool) {
	s.metaMu.RLock()
	defer s.metaMu.RUnlock()
	if s.cachedMeta != nil {
		return *s.cachedMeta, true
	}
	return session.Metadata{}, false
}

func (s *TarGzEventSource) setCachedMetadata(meta session.Metadata) {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	s.cachedMeta = &meta
}

func (s *TarGzEventSource) openTarReader() (*os.File, *gzip.Reader, *tar.Reader, error) {
	f, err := os.Open(s.archivePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrRecordingNotFound, err.Error())
	}
	gzr, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, nil, nil, fmt.Errorf("%w: open gzip reader: %s", ErrCorruptArchive, err.Error())
	}
	return f, gzr, tar.NewReader(gzr), nil
}

// LoadMetadata extracts and parses metadata.json from the tar.gz archive, caching the result.
func (s *TarGzEventSource) LoadMetadata(ctx context.Context) (session.Metadata, error) {
	if meta, ok := s.getCachedMetadata(); ok {
		return meta, nil
	}

	f, gzr, tr, err := s.openTarReader()
	if err != nil {
		return session.Metadata{}, err
	}
	defer func() {
		_ = gzr.Close()
		_ = f.Close()
	}()

	meta, err := s.findMetadataInTar(ctx, tr)
	if err != nil {
		return session.Metadata{}, err
	}
	s.setCachedMetadata(meta)
	return meta, nil
}

func (s *TarGzEventSource) findMetadataInTar(ctx context.Context, tr *tar.Reader) (session.Metadata, error) {
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

// StreamTicks streams 1-second TickBatches using memory-bounded K-Way Merge across event streams.
func (s *TarGzEventSource) StreamTicks(ctx context.Context) (<-chan events.TickBatch, <-chan error, error) {
	tmpDir, err := os.MkdirTemp("", "streamer-session-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp extraction dir: %w", err)
	}

	meta, extractedFiles, err := s.extractSessionArchive(ctx, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, nil, err
	}

	streams, err := s.openEventStreams(extractedFiles)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, nil, err
	}

	tickCh := make(chan events.TickBatch)
	errCh := make(chan error, 1)

	go s.streamKWayMerge(ctx, meta, streams, tmpDir, tickCh, errCh)

	return tickCh, errCh, nil
}

func (s *TarGzEventSource) extractSessionArchive(ctx context.Context, targetDir string) (session.Metadata, []string, error) {
	cachedMeta, metaFound := s.getCachedMetadata()

	f, gzr, tr, err := s.openTarReader()
	if err != nil {
		return session.Metadata{}, nil, err
	}
	defer func() {
		_ = gzr.Close()
		_ = f.Close()
	}()

	meta, files, err := s.processArchiveEntries(ctx, tr, targetDir, cachedMeta, metaFound)
	if err != nil {
		return session.Metadata{}, nil, err
	}
	sort.Strings(files)
	return meta, files, nil
}

func (s *TarGzEventSource) processArchiveEntries(
	ctx context.Context,
	tr *tar.Reader,
	targetDir string,
	initialMeta session.Metadata,
	metaFound bool,
) (session.Metadata, []string, error) {
	var extractedFiles []string
	meta := initialMeta

	for {
		if err := ctx.Err(); err != nil {
			return session.Metadata{}, nil, err
		}

		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return session.Metadata{}, nil, fmt.Errorf("%w: read tar entry: %s", ErrCorruptArchive, err.Error())
		}

		outPath, err := s.processSingleEntry(tr, header, targetDir, &metaFound, &meta)
		if err != nil {
			return session.Metadata{}, nil, err
		}
		if outPath != "" {
			extractedFiles = append(extractedFiles, outPath)
		}
	}

	if !metaFound {
		return session.Metadata{}, nil, ErrMetadataNotFound
	}
	return meta, extractedFiles, nil
}

func (s *TarGzEventSource) processSingleEntry(
	tr *tar.Reader,
	header *tar.Header,
	targetDir string,
	metaFound *bool,
	meta *session.Metadata,
) (string, error) {
	if strings.HasSuffix(header.Name, "metadata.json") && !*metaFound {
		m, err := s.decodeMetadataHeader(tr)
		if err != nil {
			return "", err
		}
		*meta = m
		*metaFound = true
		s.setCachedMetadata(m)
		return "", nil
	}

	return extractTarEntry(tr, header, targetDir)
}

func extractTarEntry(tr *tar.Reader, header *tar.Header, targetDir string) (string, error) {
	baseName := filepath.Base(header.Name)
	if !strings.HasSuffix(baseName, ".jsonl") {
		return "", nil
	}

	outPath := filepath.Join(targetDir, baseName)
	outFile, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create temp event file: %w", err)
	}

	if _, err := io.Copy(outFile, tr); err != nil {
		_ = outFile.Close()
		return "", fmt.Errorf("%w: write temp event file: %s", ErrCorruptArchive, err.Error())
	}
	if err := outFile.Close(); err != nil {
		return "", fmt.Errorf("close temp event file: %w", err)
	}

	return outPath, nil
}

func (s *TarGzEventSource) openEventStreams(files []string) ([]*eventStream, error) {
	var streams []*eventStream
	for _, file := range files {
		st, err := newEventStream(file, s.reporter)
		if err != nil {
			for _, prev := range streams {
				prev.close()
			}
			return nil, err
		}
		streams = append(streams, st)
	}
	return streams, nil
}

type eventStream struct {
	filename string
	file     *os.File
	scanner  *bufio.Scanner
	currEv   events.Event
	lineNum  int
	reporter ports.ErrorReporter
}

func newEventStream(path string, rep ports.ErrorReporter) (*eventStream, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	base := filepath.Base(path)
	st := &eventStream{
		filename: base,
		file:     f,
		scanner:  bufio.NewScanner(f),
		reporter: rep,
	}
	const maxScanCapacity = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	st.scanner.Buffer(buf, maxScanCapacity)
	if err := st.advance(); err != nil {
		_ = f.Close()
		return nil, err
	}
	return st, nil
}

func (st *eventStream) advance() error {
	for st.scanner.Scan() {
		st.lineNum++
		line := st.scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		ev, err := parseEventLine(st.filename, line)
		if err != nil {
			st.reporter.ReportWarning("malformed event line skipped", map[string]any{
				"filename": st.filename,
				"line":     st.lineNum,
				"error":    err.Error(),
			})
			continue
		}
		if ev != nil {
			st.currEv = ev
			return nil
		}
	}
	if err := st.scanner.Err(); err != nil {
		return fmt.Errorf("%w: scanner error in %s: %s", ErrCorruptArchive, st.filename, err.Error())
	}
	st.currEv = nil
	return nil
}

func (st *eventStream) close() {
	if st.file != nil {
		_ = st.file.Close()
	}
}

type streamHeap []*eventStream

func (h streamHeap) Len() int { return len(h) }
func (h streamHeap) Less(i, j int) bool {
	ti := h[i].currEv.GetTimestamp()
	tj := h[j].currEv.GetTimestamp()
	if !ti.Equal(tj) {
		return ti.Before(tj)
	}
	return h[i].filename < h[j].filename
}
func (h streamHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *streamHeap) Push(x any)   { *h = append(*h, x.(*eventStream)) }
func (h *streamHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func advanceMinStream(h *streamHeap) error {
	minStream := (*h)[0]
	if err := minStream.advance(); err != nil {
		return err
	}
	if minStream.currEv == nil {
		heap.Pop(h)
		minStream.close()
	} else {
		heap.Fix(h, 0)
	}
	return nil
}

func collectBatchEvents(ctx context.Context, h *streamHeap, batch *events.TickBatch, currStart, currEnd time.Time) error {
	for h.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		minStream := (*h)[0]
		ts := minStream.currEv.GetTimestamp()

		if batch.TickIndex > 0 && ts.Before(currStart) {
			if err := advanceMinStream(h); err != nil {
				return err
			}
			continue
		}

		if !ts.Before(currEnd) {
			break
		}

		if err := batch.Add(minStream.currEv); err != nil {
			return err
		}
		if err := advanceMinStream(h); err != nil {
			return err
		}
	}
	return nil
}

func initStreamHeap(streams []*eventStream) *streamHeap {
	h := &streamHeap{}
	for _, st := range streams {
		if st.currEv != nil {
			*h = append(*h, st)
		}
	}
	heap.Init(h)
	return h
}

func emitTrailingBatch(
	ctx context.Context,
	h *streamHeap,
	tickIdx int,
	currStart time.Time,
	tickCh chan<- events.TickBatch,
) error {
	if h.Len() == 0 {
		return nil
	}

	finalStart := currStart
	finalEnd := finalStart.Add(time.Second)

	batch, err := events.NewTickBatch(tickIdx, finalStart, finalEnd)
	if err != nil {
		return err
	}

	for h.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		minStream := (*h)[0]
		ts := minStream.currEv.GetTimestamp()
		if !ts.Before(batch.EndTime) {
			batch.EndTime = ts.Add(time.Second)
		}

		if err := batch.Add(minStream.currEv); err != nil {
			return err
		}
		if err := advanceMinStream(h); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case tickCh <- batch:
		return nil
	}
}

func (s *TarGzEventSource) streamKWayMerge(
	ctx context.Context,
	meta session.Metadata,
	streams []*eventStream,
	tmpDir string,
	tickCh chan<- events.TickBatch,
	errCh chan<- error,
) {
	defer close(tickCh)
	defer close(errCh)
	defer func() {
		for _, st := range streams {
			st.close()
		}
		_ = os.RemoveAll(tmpDir)
	}()

	h := initStreamHeap(streams)
	currStart := meta.TimeRange.Start
	endedAt := meta.TimeRange.End
	tickIdx := 0

	for currStart.Before(endedAt) {
		currEnd := currStart.Add(time.Second)
		if currEnd.After(endedAt) {
			currEnd = endedAt
		}

		batch, err := events.NewTickBatch(tickIdx, currStart, currEnd)
		if err != nil {
			errCh <- err
			return
		}

		if err := collectBatchEvents(ctx, h, &batch, currStart, currEnd); err != nil {
			errCh <- err
			return
		}

		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		case tickCh <- batch:
		}

		currStart = currEnd
		tickIdx++
	}

	if err := emitTrailingBatch(ctx, h, tickIdx, currStart, tickCh); err != nil {
		errCh <- err
	}
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
