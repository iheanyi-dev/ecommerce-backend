package ports

import "context"

// UnitOfWork commits subscription changes and outbox events atomically.
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}
