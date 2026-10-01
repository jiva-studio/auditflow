package telemetry

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *responseWriterInterceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

// TraceMiddleware returns a standard net/http middleware that extracts or generates a trace ID,
// attaches a contextual slog.Logger to the request context, sets the X-Trace-ID response header,
// and records HTTP request telemetry.
func TraceMiddleware(serviceName string, baseLogger *slog.Logger) func(http.Handler) http.Handler {
	if baseLogger == nil {
		baseLogger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := strings.TrimSpace(r.Header.Get(HeaderTraceID))
			if traceID == "" {
				traceID = GenerateTraceID()
			}

			w.Header().Set(HeaderTraceID, traceID)

			reqLogger := baseLogger.With(
				slog.String("trace_id", traceID),
				slog.String("service", serviceName),
			)

			ctx := WithTraceID(r.Context(), traceID)
			ctx = WithLogger(ctx, reqLogger)

			start := time.Now()
			interceptor := &responseWriterInterceptor{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(interceptor, r.WithContext(ctx))

			duration := time.Since(start)

			// Suppress excessive healthcheck logs if status is 200 OK
			if strings.HasPrefix(r.URL.Path, "/health") && interceptor.statusCode == http.StatusOK {
				return
			}

			reqLogger.InfoContext(ctx, "HTTP request handled",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", interceptor.statusCode),
				slog.Int64("bytes", interceptor.bytesWritten),
				slog.Duration("duration_ms", duration),
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}
