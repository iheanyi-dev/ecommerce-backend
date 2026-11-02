package postgres

import (
	"context"
	"fmt"

	db "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transactionContextKey struct{}

// WithTx makes a PostgreSQL transaction available to repositories through context.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

func queriesFor(ctx context.Context, pool *pgxpool.Pool) (*db.Queries, error) {
	if pool == nil {
		return nil, fmt.Errorf("postgres connection pool is required")
	}

	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok && tx != nil {
		return db.New(tx), nil
	}

	return db.New(pool), nil
}
