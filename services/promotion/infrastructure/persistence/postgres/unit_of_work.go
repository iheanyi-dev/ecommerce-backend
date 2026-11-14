package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) (*UnitOfWork, error) {
	if pool == nil {
		return nil, fmt.Errorf("promotion unit of work requires a database pool")
	}
	return &UnitOfWork{pool: pool}, nil
}

func (u *UnitOfWork) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	if u == nil || u.pool == nil {
		return fmt.Errorf("promotion unit of work requires a database pool")
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
		return fmt.Errorf("begin promotion transaction: %w", err)
	}

	finished := false
	defer func() {
		if finished {
			return
		}
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	txCtx := WithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		rollbackErr := tx.Rollback(context.Background())
		finished = true
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("rollback promotion transaction: %w", rollbackErr))
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		finished = true
		return fmt.Errorf("commit promotion transaction: %w", err)
	}

	finished = true
	return nil
}

var _ ports.UnitOfWork = (*UnitOfWork)(nil)
