package infrastructure

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/migrations"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

// Module provides Store infrastructure dependencies.
//
// Infrastructure implementations satisfy application-layer ports while
// keeping the application layer independent of PostgreSQL, migrations,
// OpenTelemetry, logging, and metrics implementations.
var Module = fx.Module(
	"infrastructure",

	fx.Provide(
		config.Load,

		postgres.NewPool,
		postgres.NewQueries,

		fx.Annotate(
			postgres.NewStoreRepository,
		),

		migrations.NewMigrationRunner,

		// Provide the concrete logger and expose it through the
		// application-layer Logger port.
		fx.Annotate(
			observability.NewProductionLogger,
			fx.As(new(ports.Logger)),
		),

		// Store currently uses an in-memory metrics implementation.
		// The application layer depends only on the Metrics port, so this
		// implementation can later be replaced by Prometheus/OpenTelemetry
		// metrics without changing application code.
		fx.Annotate(
			observability.NewMetrics,
			fx.As(new(ports.Metrics)),
		),

		// The Store HTTP tracing middleware depends only on the application
		// Tracer port. OpenTelemetry remains entirely inside infrastructure.
		fx.Annotate(
			newStoreTracer,
			fx.As(new(ports.Tracer)),
		),
	),

	fx.Invoke(
		postgres.RegisterLifecycle,
		migrations.RegisterLifecycle,
	),
)

// newStoreTracer creates the Store's infrastructure tracing adapter.
//
// The service name is deliberately fixed at the composition boundary:
// it identifies the service in telemetry and is not business configuration.
func newStoreTracer() (*observability.OpenTelemetryTracer, error) {
	return observability.NewOpenTelemetryTracer("store")
}
