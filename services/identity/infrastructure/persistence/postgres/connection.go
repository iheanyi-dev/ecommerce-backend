package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// NewPool creates a PostgreSQL connection pool.
//
// The pool is created here rather than inside the repository because
// connection management is an infrastructure concern.
//
// Fx provides the application configuration. The connection pool creates
// its own initialization context because context.Context is request- or
// operation-scoped and should not be treated as a global dependency.
func NewPool(
	cfg *config.Config,
) (*pgxpool.Pool, error) {

	// Give PostgreSQL connection initialization a bounded amount of time.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(
		cfg.DatabaseURL(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse database configuration: %w",
			err,
		)
	}

	// These values control how the application communicates with PostgreSQL.
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(
		ctx,
		poolConfig,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create postgres connection pool: %w",
			err,
		)
	}

	// Creating a pool does not necessarily prove that PostgreSQL is
	// reachable. Ping verifies the connection immediately.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf(
			"ping postgres: %w",
			err,
		)
	}

	return pool, nil
}