package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/persistence/postgres/generated"
)

// UnitOfWork coordinates order and product-snapshot writes in one transaction.
type UnitOfWork struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork creates a unit of work backed by the supplied PostgreSQL pool.
func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

// WithinTransaction commits all callback writes together or rolls them all back.
// Repositories supplied to the callback share the same transaction-bound queries.
func (u *UnitOfWork) WithinTransaction(
	ctx context.Context,
	fn func(ports.OrderRepository, ports.OrderProductRepository) error,
) error {
	if fn == nil {
		return errors.New("order unit of work callback cannot be nil")
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin order transaction: %w", err)
	}

	// A background context ensures rollback is attempted even if the request
	// context was canceled. After a successful commit, this rollback is harmless.
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	queries := generated.New(tx)
	orderRepository := newOrderRepository(queries)
	productRepository := newOrderProductRepository(queries)

	if err := fn(orderRepository, productRepository); err != nil {
		return fmt.Errorf("execute order transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit order transaction: %w", err)
	}

	return nil
}

// Compile-time check keeps the adapter aligned with the application contract.
var _ ports.OrderUnitOfWork = (*UnitOfWork)(nil)

// Ensure the transaction API used here remains pgx-compatible.
var _ pgx.Tx = (pgx.Tx)(nil)
