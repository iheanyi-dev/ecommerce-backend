package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// RegisterLifecycle registers database pool cleanup with Fx.
//
// The pool is a process-level resource and must be closed during normal
// application shutdown.
func RegisterLifecycle(lc fx.Lifecycle, pool *pgxpool.Pool) {
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			pool.Close()
			return nil
		},
	})
}
