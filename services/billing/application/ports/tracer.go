package ports

import "context"

// Span represents an active application trace span.
type Span interface {
	End()
}

// Tracer starts application trace spans without exposing a tracing vendor.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
