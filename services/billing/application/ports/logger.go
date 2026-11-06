package ports

import "context"

// LogEvent contains structured, non-sensitive information about an application event.
type LogEvent struct {
	Event           string
	Operation       string
	FailureCategory string
}

// Logger records application events.
// Implementations must not be allowed to affect business outcomes.
type Logger interface {
	Log(ctx context.Context, event LogEvent) error
}
