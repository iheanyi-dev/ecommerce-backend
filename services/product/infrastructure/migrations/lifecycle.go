package migrations

import (
	"context"

	"go.uber.org/fx"
)

// RegisterLifecycle runs database migrations during startup.
func RegisterLifecycle(lc fx.Lifecycle, runner *MigrationRunner) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return runner.Run(ctx)
		},
	})
}
