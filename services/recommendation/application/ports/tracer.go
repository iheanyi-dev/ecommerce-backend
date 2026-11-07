package ports

import "context"

// Span represents the minimal tracing information required by the application.
type Span interface {
	End()
}

// Tracer creates and manages application tracing spans.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
