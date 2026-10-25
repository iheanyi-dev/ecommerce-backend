package infrastructure

import (
	cartinfra "github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/cart"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
)

// Module provides Order infrastructure dependencies.
//
// PostgreSQL repositories and the unit of work implement application-layer
// ports, keeping application use cases independent of database details.
var Module = fx.Module(
	"order-infrastructure",

	fx.Provide(
		fx.Annotate(cartinfra.NewClient, fx.As(new(ports.CartClient))),
		config.Load,
		postgres.NewPool,
		fx.Annotate(
			observability.NewProductionLogger,
			fx.As(new(ports.Logger)),
		),
		fx.Annotate(
			observability.NewMetrics,
			fx.As(new(ports.Metrics)),
		),
		fx.Annotate(
			newOrderTracer,
			fx.As(new(ports.Tracer)),
		),
	),

	fx.Provide(
		fx.Annotate(
			postgres.NewOrderRepository,
			fx.As(new(ports.OrderRepository)),
		),
		fx.Annotate(
			postgres.NewOrderProductRepository,
			fx.As(new(ports.OrderProductRepository)),
		),
		fx.Annotate(
			postgres.NewUnitOfWork,
			fx.As(new(ports.OrderUnitOfWork)),
		),
	),

	fx.Invoke(postgres.RegisterLifecycle),
)

// newOrderTracer creates the infrastructure tracing adapter.
func newOrderTracer() (*observability.OpenTelemetryTracer, error) {
	return observability.NewOpenTelemetryTracer("order")
}
