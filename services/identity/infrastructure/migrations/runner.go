package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

const migrationDirectory = "services/identity/migrations"

// MigrationRunner runs database migrations during Identity service startup.
//
// Migrations are infrastructure concerns and therefore remain outside the
// application and domain layers.
type MigrationRunner struct {
	cfg *config.Config
}

// NewMigrationRunner creates the Identity database migration runner.
func NewMigrationRunner(
	cfg *config.Config,
) *MigrationRunner {
	return &MigrationRunner{
		cfg: cfg,
	}
}

// RegisterLifecycle registers migration execution with the Fx lifecycle.
//
// Migrations run during OnStart. If migration execution fails, Fx startup
// fails and the HTTP server is not started.
func RegisterLifecycle(
	lc fx.Lifecycle,
	runner *MigrationRunner,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return runner.Run(ctx)
		},
	})
}

// Run executes all pending database migrations.
func (r *MigrationRunner) Run(ctx context.Context) error {
	db, err := sql.Open(
		"pgx",
		r.cfg.DatabaseURL(),
	)
	if err != nil {
		return fmt.Errorf(
			"open migration database connection: %w",
			err,
		)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(
		ctx,
		10*time.Second,
	)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf(
			"ping migration database: %w",
			err,
		)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf(
			"set goose dialect: %w",
			err,
		)
	}

	if err := goose.UpContext(
		ctx,
		db,
		migrationDirectory,
	); err != nil {
		return fmt.Errorf(
			"run database migrations: %w",
			err,
		)
	}

	return nil
}
