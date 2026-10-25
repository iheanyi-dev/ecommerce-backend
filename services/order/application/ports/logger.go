package ports

import "context"

type LogEvent struct {
	Event           string
	Operation       string
	UserID          string
	Role            string
	FailureCategory string
}

type Logger interface {
	Log(ctx context.Context, event LogEvent) error
}
