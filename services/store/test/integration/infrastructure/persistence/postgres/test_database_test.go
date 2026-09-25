package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

// TestDatabase provides a real PostgreSQL connection pool for Store
// integration tests.
//
// These tests intentionally exercise the actual PostgreSQL schema and SQLC
// generated queries rather than replacing persistence with mocks.
type TestDatabase struct {
	Pool *pgxpool.Pool
}

// NewTestDatabase creates a PostgreSQL pool using the Store service
// configuration.
//
// Integration tests are expected to run against the PostgreSQL instance
// configured for the Store service. Tests fail immediately when the database
// is unavailable rather than silently becoming unit tests.
func NewTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load store test config: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("parse store database URL: %v", err)
	}

	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create store test database pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping store test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return &TestDatabase{Pool: pool}
}

// BeginTx starts a transaction and automatically rolls it back when the test
// finishes.
//
// Each repository integration test therefore runs in isolation without
// permanently modifying the shared Store test database.
func (db *TestDatabase) BeginTx(t *testing.T) pgx.Tx {
	t.Helper()

	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin store test transaction: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})

	return tx
}

// TestMain loads the repository-level environment file before any Store
// integration test attempts to construct its database configuration.
func TestMain(m *testing.M) {
	// Integration tests explicitly load the Store environment before any test
	// constructs the database configuration. This mirrors the established
	// Identity integration-test convention and keeps production configuration
	// loading independent from test setup.
	_ = godotenv.Load("../../../../../.env")
	_ = godotenv.Load("services/store/.env")

	_ = os.Setenv("GOOSE_DRIVER", "postgres")

	os.Exit(m.Run())
}
