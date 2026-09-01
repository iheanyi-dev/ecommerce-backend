package postgres_tests

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// TestDatabase provides PostgreSQL resources for integration tests.
//
// The tests use the real Identity PostgreSQL database. Individual tests
// execute inside transactions which are rolled back after the test.
//
// This means test data does not remain in the database after a test.
type TestDatabase struct {
	pool *pgxpool.Pool
}

// NewTestDatabase creates a PostgreSQL connection pool for integration tests.
func NewTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf(
			"failed to load database configuration: %v",
			err,
		)
	}

	poolConfig, err := pgxpool.ParseConfig(
		cfg.DatabaseURL(),
	)
	if err != nil {
		t.Fatalf(
			"failed to parse database configuration: %v",
			err,
		)
	}

	pool, err := pgxpool.NewWithConfig(
		context.Background(),
		poolConfig,
	)
	if err != nil {
		t.Fatalf(
			"failed to create PostgreSQL connection pool: %v",
			err,
		)
	}

	// Verify that PostgreSQL is reachable.
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()

		t.Fatalf(
			"failed to ping PostgreSQL: %v",
			err,
		)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return &TestDatabase{
		pool: pool,
	}
}

// BeginTx starts a transaction for an individual test.
//
// The transaction is automatically rolled back when the test completes.
func (db *TestDatabase) BeginTx(t *testing.T) pgx.Tx {
	t.Helper()

	tx, err := db.pool.Begin(context.Background())
	if err != nil {
		t.Fatalf(
			"failed to begin test transaction: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})

	return tx
}