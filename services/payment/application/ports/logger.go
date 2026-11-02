package ports

import "context"

// LogEvent represents a safe, structured application event.
type LogEvent struct {
	Event           string
	Operation       string
	UserID          string
	Role            string
	FailureCategory string
}

// Logger records application events. Logging failures must not change
// the business operation's result.
type Logger interface {
	Log(ctx context.Context, event LogEvent) error
}
