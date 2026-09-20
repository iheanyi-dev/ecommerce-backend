package ports

import "context"

// Span represents the minimal tracing information required by the Store
// application boundary.
//
// The application layer deliberately does not depend on OpenTelemetry or
// another tracing implementation.
type Span interface {
	End()
}

// Tracer creates and manages Store application tracing spans.
//
// Concrete tracing concerns remain in Infrastructure. The application
// depends only on this small abstraction so the tracing implementation can
// be replaced without changing Store business logic.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
