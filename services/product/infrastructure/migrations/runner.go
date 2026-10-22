package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

const migrationDirectory = "services/product/migrations"

// MigrationRunner applies pending Product database migrations.
type MigrationRunner struct {
	config *config.Config
}

// NewMigrationRunner creates the Product migration runner.
func NewMigrationRunner(cfg *config.Config) *MigrationRunner {
	return &MigrationRunner{config: cfg}
}

// Run applies all pending migrations to the Product database.
func (r *MigrationRunner) Run(ctx context.Context) error {
	if r.config == nil {
		return fmt.Errorf("product migration runner: config is nil")
	}

	db, err := sql.Open("pgx", r.config.DatabaseURL())
	if err != nil {
		return fmt.Errorf("open product migration database: %w", err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping product migration database: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configure product goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, migrationDirectory); err != nil {
		return fmt.Errorf("run product migrations: %w", err)
	}

	return nil
}
