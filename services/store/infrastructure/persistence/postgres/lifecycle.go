package postgres

import (
	"context"

	"go.uber.org/fx"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterLifecycle registers PostgreSQL resource cleanup with Fx.
//
// The connection pool is a process-level resource and must be closed when
// the Store service shuts down. Fx guarantees that this cleanup participates
// in the application's normal lifecycle.
func RegisterLifecycle(
	lc fx.Lifecycle,
	pool *pgxpool.Pool,
) {
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			pool.Close()

			return nil
		},
	})
}
