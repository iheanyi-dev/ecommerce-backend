package postgres

import (
	"context"
	"fmt"

	paymentmigrations "github.com/iheanyi-dev/ecommerce-backend/services/payment/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
)

const migrationsDirectory = "."

// RegisterMigrations runs pending migrations before the Payment service starts.
func RegisterMigrations(
	lifecycle fx.Lifecycle,
	pool *pgxpool.Pool,
) error {
	if lifecycle == nil {
		return fmt.Errorf("payment migration lifecycle is required")
	}
	if pool == nil {
		return fmt.Errorf("payment database pool is required for migrations")
	}

	db := stdlib.OpenDBFromPool(pool)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := goose.SetDialect("postgres"); err != nil {
				return fmt.Errorf("configure payment migration dialect: %w", err)
			}

			goose.SetBaseFS(paymentmigrations.FS)

			if err := goose.UpContext(ctx, db, migrationsDirectory); err != nil {
				return fmt.Errorf("run payment database migrations: %w", err)
			}

			return nil
		},
		OnStop: func(context.Context) error {
			return db.Close()
		},
	})

	return nil
}
