package ports

import "context"

// LogEvent describes a safe, structured application event.
//
// Business identifiers and object contents must not be placed in logs.
type LogEvent struct {
	Event           string
	Operation       string
	FailureCategory string
}

// Logger provides application-level structured logging.
type Logger interface {
	Log(ctx context.Context, event LogEvent) error
}
