package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
)

// NewPool creates the PostgreSQL connection pool used by Order.
//
// The pool is owned by infrastructure. Application use cases interact with
// persistence through repository and unit-of-work ports instead of accessing
// the database pool directly.
func NewPool(cfg *config.Config) (*pgxpool.Pool, error) {
	// Parse the configured URL at startup so invalid connection settings fail
	// during dependency construction rather than on the first request.
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf(
			"parse order database configuration: %w",
			err,
		)
	}

	// Bound database resource usage and align pool behavior with Store.
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// Bound initialization time so a database outage cannot indefinitely
	// block service construction.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf(
			"create order database connection pool: %w",
			err,
		)
	}

	// Verify database connectivity during startup.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf(
			"ping order database: %w",
			err,
		)
	}

	return pool, nil
}
