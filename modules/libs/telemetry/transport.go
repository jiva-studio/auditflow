package telemetry

import (
	"net/http"
	"strings"
)

// TraceRoundTripper wraps an existing http.RoundTripper and injects the X-Trace-ID header
// into all outbound HTTP requests using the trace ID extracted from the request context.
type TraceRoundTripper struct {
	base http.RoundTripper
}

// NewTraceRoundTripper constructs a new TraceRoundTripper wrapping the given base transport.
func NewTraceRoundTripper(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &TraceRoundTripper{base: base}
}

// RoundTrip executes a single HTTP transaction, injecting HeaderTraceID if present in Context.
func (t *TraceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	traceID := TraceIDFromContext(req.Context())
	if strings.TrimSpace(traceID) == "" {
		traceID = GenerateTraceID()
	}

	// Clone request to avoid mutating caller's original request headers concurrently
	clonedReq := req.Clone(req.Context())
	if strings.TrimSpace(clonedReq.Header.Get(HeaderTraceID)) == "" {
		clonedReq.Header.Set(HeaderTraceID, traceID)
	}

	return t.base.RoundTrip(clonedReq)
}
