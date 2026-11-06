package postgres

import (
	"context"
	"database/sql"
	"fmt"

	billingmigrations "github.com/iheanyi-dev/ecommerce-backend/services/billing/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
)

func RegisterMigrations(lifecycle fx.Lifecycle, pool *pgxpool.Pool) error {
	if lifecycle == nil || pool == nil {
		return fmt.Errorf("Billing migration lifecycle and database pool are required")
	}

	var db *sql.DB

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			db = stdlib.OpenDBFromPool(pool)

			if err := goose.SetDialect("postgres"); err != nil {
				return fmt.Errorf("configure Billing migration dialect: %w", err)
			}

			goose.SetBaseFS(billingmigrations.FS)
			if err := goose.UpContext(ctx, db, "."); err != nil {
				_ = db.Close()
				return fmt.Errorf("run Billing database migrations: %w", err)
			}
			return nil
		},
		OnStop: func(context.Context) error {
			if db != nil {
				return db.Close()
			}
			return nil
		},
	})

	return nil
}
