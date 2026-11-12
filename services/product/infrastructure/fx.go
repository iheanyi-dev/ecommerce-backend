// Package infrastructure wires Product infrastructure adapters.
package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	productgrpc "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/grpc"
	kafkainfra "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/messaging/kafka"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/migrations"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/storage"
	storeinfra "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/store"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

// Module provides PostgreSQL persistence and migration lifecycle management.
var Module = fx.Module(
	"product-infrastructure",
	fx.Provide(
		fx.Annotate(observability.NewMetrics, fx.As(new(ports.Metrics))),
		fx.Annotate(observability.NewProductionLogger, fx.As(new(ports.Logger))),
		config.Load,
		postgres.NewPool,
		postgres.NewQueries,
		fx.Annotate(
			postgres.NewProductRepository,
			fx.As(new(ports.ProductRepository)),
		),
		fx.Annotate(
			postgres.NewProductPromotionRepository,
			fx.As(new(ports.ProductPromotionRepository)),
		),
		migrations.NewMigrationRunner,
		fx.Annotate(storage.NewClient, fx.As(new(ports.StorageClient))),
		fx.Annotate(storeinfra.NewClient, fx.As(new(ports.StoreClient))),
		fx.Annotate(kafkainfra.NewClient, fx.As(new(kafkainfra.MessagePublisher))),
		fx.Annotate(kafkainfra.NewEmbeddingDispatcher, fx.As(new(ports.EmbeddingDispatcher))),
	),
	fx.Provide(productgrpc.NewServerWithCartLookup),
	fx.Invoke(
		registerPoolLifecycle,
		migrations.RegisterLifecycle,
	),
)

func registerPoolLifecycle(lc fx.Lifecycle, pool *pgxpool.Pool) {
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})
}
