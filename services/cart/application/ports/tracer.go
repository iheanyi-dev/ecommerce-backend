package ports

import "context"

// Span is the application-facing abstraction for a traced operation.
type Span interface {
	SetAttribute(key, value string)
	SetIntAttribute(key string, value int)
	RecordError(err error)
	SetErrorStatus(description string)
	End()
}

// Tracer starts spans without exposing an observability vendor to the application.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}
