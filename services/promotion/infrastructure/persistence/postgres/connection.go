package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
)

const (
	maxPoolConnections = int32(10)
	minPoolConnections = int32(2)
	poolMaxLifetime    = time.Hour
	poolMaxIdleTime    = 30 * time.Minute
	poolHealthInterval = time.Minute
	poolInitTimeout    = 5 * time.Second
)

// NewPool creates and manages the Promotion PostgreSQL connection pool.
func NewPool(lifecycle fx.Lifecycle, cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("promotion config is required")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("promotion lifecycle is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("parse promotion database configuration: %w", err)
	}

	poolConfig.MaxConns = maxPoolConnections
	poolConfig.MinConns = minPoolConnections
	poolConfig.MaxConnLifetime = poolMaxLifetime
	poolConfig.MaxConnIdleTime = poolMaxIdleTime
	poolConfig.HealthCheckPeriod = poolHealthInterval

	ctx, cancel := context.WithTimeout(context.Background(), poolInitTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create promotion database pool: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pool.Ping(ctx); err != nil {
				pool.Close()
				return fmt.Errorf("connect to promotion database: %w", err)
			}
			return nil
		},
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}
