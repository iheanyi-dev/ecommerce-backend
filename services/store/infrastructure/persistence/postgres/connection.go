package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

// NewPool creates the PostgreSQL connection pool used by the Store service.
//
// The Store service owns its own PostgreSQL database. The application layer
// never receives the pool directly; Infrastructure uses it to construct
// concrete persistence implementations.
func NewPool(cfg *config.Config) (*pgxpool.Pool, error) {
	// Parse the configured PostgreSQL URL first so invalid connection
	// configuration fails during service construction rather than later when
	// the first request reaches the repository.
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf(
			"parse store database configuration: %w",
			err,
		)
	}

	// Keep pool behaviour explicit and predictable.
	//
	// These values mirror the existing Identity service baseline and provide
	// bounded database resources for the Store service.
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// Use a bounded initialization context so a database outage cannot leave
	// service construction waiting indefinitely.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(
		ctx,
		poolConfig,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create store database connection pool: %w",
			err,
		)
	}

	// Verify connectivity during startup. This makes database availability
	// part of Store service readiness instead of allowing the service to
	// start successfully while persistence is unavailable.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf(
			"ping store database: %w",
			err,
		)
	}

	return pool, nil
}
