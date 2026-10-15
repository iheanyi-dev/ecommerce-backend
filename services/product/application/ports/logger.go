package ports

import "context"

// LogEvent contains structured, low-risk operational context.
type LogEvent struct {
	Event           string
	Operation       string
	FailureCategory string
}

// Logger records application events. Logging failure must not change the
// business operation's result.
type Logger interface {
	Log(ctx context.Context, event LogEvent) error
}
