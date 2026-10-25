package ports

import "context"

// OrderUnitOfWork executes all order and order-line writes in one transaction.
// If the callback returns an error, the implementation must roll back.
type OrderUnitOfWork interface {
	WithinTransaction(
		ctx context.Context,
		fn func(OrderRepository, OrderProductRepository) error,
	) error
}
