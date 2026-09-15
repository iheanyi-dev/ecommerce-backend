package ports

import "context"

// Span represents the minimal tracing information that the application
// boundary needs to propagate across operations.
//
// The application layer deliberately does not depend on OpenTelemetry or
// another tracing implementation. Concrete tracing concerns remain in the
// infrastructure layer.
type Span interface {
	End()
}

// Tracer creates and manages tracing spans.
//
// Implementations are responsible for propagating trace context through the
// supplied context. The application depends only on this interface so the
// tracing implementation can later be replaced without changing business
// logic.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
