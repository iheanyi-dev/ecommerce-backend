package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const rollbackTimeout = 5 * time.Second

type UnitOfWork struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork creates a PostgreSQL unit of work.
func NewUnitOfWork(pool *pgxpool.Pool) (*UnitOfWork, error) {
	if pool == nil {
		return nil, fmt.Errorf("unit of work requires a database pool")
	}
	return &UnitOfWork{pool: pool}, nil
}

// WithinTransaction executes fn atomically.
func (u *UnitOfWork) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	if u == nil || u.pool == nil {
		return fmt.Errorf("unit of work requires a database pool")
	}
	if ctx == nil {
		return fmt.Errorf("transaction context is required")
	}
	if fn == nil {
		return fmt.Errorf("transaction callback is required")
	}

	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok && tx != nil {
		return fn(ctx)
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin billing transaction: %w", err)
	}

	txCtx := WithTx(ctx, tx)
	transactionFinished := false

	defer func() {
		if transactionFinished {
			return
		}

		rollbackCtx, cancel := context.WithTimeout(
			context.Background(),
			rollbackTimeout,
		)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	if err := fn(txCtx); err != nil {
		rollbackCtx, cancel := context.WithTimeout(
			context.Background(),
			rollbackTimeout,
		)
		rollbackErr := tx.Rollback(rollbackCtx)
		cancel()
		transactionFinished = true

		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(
				fmt.Errorf("execute billing transaction: %w", err),
				fmt.Errorf("rollback billing transaction: %w", rollbackErr),
			)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		transactionFinished = true
		return fmt.Errorf("commit billing transaction: %w", err)
	}

	transactionFinished = true
	return nil
}

var _ ports.UnitOfWork = (*UnitOfWork)(nil)
