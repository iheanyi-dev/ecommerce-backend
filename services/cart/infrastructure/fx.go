// Package infrastructure wires Cart infrastructure adapters.
package infrastructure

import (
	"context"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/observability"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	redisinfra "github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/persistence/redis"
	productinfra "github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/product"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/shared/config"
)

// Module provides Redis persistence and connection lifecycle management.
var Module = fx.Module(
	"cart-infrastructure",
	fx.Provide(
		fx.Annotate(observability.NewTracer, fx.As(new(ports.Tracer))),
		fx.Annotate(observability.NewProductionLogger, fx.As(new(ports.Logger))),
		fx.Annotate(observability.NewMetrics, fx.As(new(ports.Metrics))),
		config.Load,
		NewRedisClient,
		fx.Annotate(
			redisinfra.NewCartRepository,
			fx.As(new(ports.CartRepository)),
		),
		fx.Annotate(
			productinfra.NewClient,
			fx.As(new(ports.ProductClient)),
		),
	),
	fx.Invoke(RegisterRedisLifecycle),
)

// NewRedisClient creates the shared Redis client used by Cart.
func NewRedisClient(cfg *config.Config) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}

// RegisterRedisLifecycle checks Redis connectivity during startup and closes
// the client when the Fx application stops.
func RegisterRedisLifecycle(lc fx.Lifecycle, client *goredis.Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return client.Ping(ctx).Err()
		},
		OnStop: func(context.Context) error {
			return client.Close()
		},
	})
}
