package ports

import "context"

// Span represents the minimal tracing information required by the
// Promotion HTTP presentation boundary.
type Span interface {
	End()
}

// Tracer creates HTTP request tracing spans.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
