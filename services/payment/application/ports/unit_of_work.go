package ports

import "context"

// UnitOfWork commits payment and attempt changes atomically.
type UnitOfWork interface {
	// WithinTransaction runs fn in a transaction and commits only if fn succeeds.
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}
