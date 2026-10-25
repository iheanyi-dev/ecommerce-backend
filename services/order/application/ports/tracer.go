package ports

import "context"

// Span is the minimum tracing contract required by the application layer.
type Span interface {
	End()
}

// Tracer starts spans without exposing a tracing vendor to the application.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
