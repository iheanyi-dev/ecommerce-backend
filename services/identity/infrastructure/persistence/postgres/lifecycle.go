package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// RegisterPoolLifecycle registers the PostgreSQL connection pool with the
// Fx application lifecycle.
//
// Fx will close the pool automatically when the application shuts down.
func RegisterPoolLifecycle(
	lc fx.Lifecycle,
	pool *pgxpool.Pool,
) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})
}
