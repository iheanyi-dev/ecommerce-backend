package ports

import "context"

// Span represents the minimal tracing information required by the
// Payment application boundary.
//
// The application layer deliberately does not depend on OpenTelemetry
// or another tracing implementation.
type Span interface {
	End()
}

// Tracer creates and manages Payment application tracing spans.
//
// Concrete tracing concerns remain in Infrastructure.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
