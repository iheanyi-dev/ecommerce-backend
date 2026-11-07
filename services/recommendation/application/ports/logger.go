package ports

import "context"

// LogEvent represents a structured Recommendation application event.
type LogEvent struct {
	Event           string
	Operation       string
	FailureCategory string
}

// Logger defines the application boundary for structured logging.
type Logger interface {
	Log(ctx context.Context, event LogEvent) error
}
