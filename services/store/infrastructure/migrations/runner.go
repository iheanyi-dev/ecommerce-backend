package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pressly/goose/v3"
	"go.uber.org/fx"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

// migrationDirectory is deliberately relative to the repository root.
//
// The Store service is currently part of the root Go module, so the
// migration runner uses the same repository-relative convention as the
// Identity service.
const migrationDirectory = "services/store/migrations"

// MigrationRunner owns execution of Store database migrations.
//
// Migrations run against the Store service database only. The Store service
// deliberately has its own persistence boundary and does not share database
// tables with other services.
type MigrationRunner struct {
	config *config.Config
}

// NewMigrationRunner constructs the Store migration runner.
func NewMigrationRunner(cfg *config.Config) *MigrationRunner {
	return &MigrationRunner{
		config: cfg,
	}
}

// Run applies all pending Store migrations.
//
// A fresh database is migrated to the latest schema before the service begins
// serving requests. Goose tracks applied migrations in its own version table.
func (r *MigrationRunner) Run(ctx context.Context) error {
	if r.config == nil {
		return fmt.Errorf("store migration runner: config is nil")
	}

	db, err := sql.Open("pgx", r.config.DatabaseURL())
	if err != nil {
		return fmt.Errorf("open store migration database: %w", err)
	}
	defer db.Close()

	// Give database initialization a bounded amount of time so a startup
	// failure cannot leave the service waiting indefinitely.
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping store migration database: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configure store goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, migrationDirectory); err != nil {
		return fmt.Errorf("run store migrations: %w", err)
	}

	return nil
}

// RegisterLifecycle runs Store migrations during Fx startup.
//
// Migrations are intentionally executed before the HTTP service should begin
// accepting traffic, ensuring the persistence schema is ready first.
func RegisterLifecycle(lc fx.Lifecycle, runner *MigrationRunner) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return runner.Run(ctx)
		},
	})
}
