// Package ports defines the inbound and outbound interfaces for the streamer service.
package ports

// ErrorReporter defines the port for reporting streaming anomalies, malformed records, and warnings.
type ErrorReporter interface {
	ReportError(err error, context map[string]any)
	ReportWarning(msg string, context map[string]any)
}
