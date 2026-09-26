package infrastructure

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	identityinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/identity"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/image"
	kafkainfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/kafka"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/migrations"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres"
	storageinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/storage"
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
			image.NewImageFormatResolver,
			fx.As(new(ports.ImageFormatResolver)),
		),

		fx.Annotate(
			identityinfra.NewClient,
		),

		fx.Annotate(
			identityinfra.NewProvider,
			fx.As(new(ports.IdentityProvider)),
		),

		// Store's synchronous image operations use Storage through the
		// application-layer ImageStorage port.
		fx.Annotate(
			storageinfra.NewClient,
			fx.As(new(ports.ImageStorage)),
		),

		// Kafka producer used by Store asynchronous image workflows.
		fx.Annotate(
			kafkainfra.NewClient,
			fx.As(new(kafkainfra.MessagePublisher)),
		),

		fx.Annotate(
			kafkainfra.NewImageDispatcher,
			fx.As(new(ports.ImageStorageDispatcher)),
		),

		fx.Annotate(
			kafkainfra.NewImageDeletionDispatcher,
			fx.As(new(ports.StoreImageDeletionDispatcher)),
		),

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
