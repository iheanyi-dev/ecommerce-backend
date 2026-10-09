package ports

import "context"

// Logger records structured application events.
//
// The interface keeps Gateway application and presentation code independent
// from the concrete logging implementation.
type Logger interface {
	Info(ctx context.Context, message string, fields ...Field)
	Error(ctx context.Context, message string, fields ...Field)
}

// Field represents one structured logging attribute.
type Field struct {
	Key   string
	Value any
}
