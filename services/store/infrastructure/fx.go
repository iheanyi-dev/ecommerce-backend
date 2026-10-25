package infrastructure

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	chatinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/chat"
	storegrpc "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/grpc"
	identityinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/identity"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/image"
	kafkainfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/kafka"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/migrations"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/observability"
	orderinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/order"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres"
	productinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/product"
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

	fx.Provide(fx.Annotate(productinfra.NewClient, fx.As(new(ports.ProductCountProvider)))),
	fx.Provide(fx.Annotate(orderinfra.NewClient, fx.As(new(ports.OrderClient)))),
	fx.Provide(
		fx.Annotate(
			chatinfra.NewClient,
			fx.As(new(ports.ChatDispatcher)),
		),
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
			identityinfra.NewConfiguredProvider,
			fx.As(new(ports.IdentityProvider)),
		),

		// Store's synchronous image operations use Storage through the
		// application-layer ImageStorage port.
		fx.Annotate(
			storageinfra.NewClient,
			fx.As(new(ports.ImageStorage)),
			fx.As(new(ports.KnowledgeStorage)),
			fx.As(new(ports.KnowledgeFileDeleter)),
		),

		// Recommendation embedding work is asynchronous. Store publishes
		// embedding events to Kafka; Recommendation consumes them.
		fx.Annotate(
			kafkainfra.NewEmbeddingDispatcher,
			fx.As(new(ports.EmbeddingDispatcher)),
		),

		fx.Annotate(
			kafkainfra.NewKnowledgeDispatcher,
			fx.As(new(ports.KnowledgeIngestionDispatcher)),
		),
		fx.Annotate(
			kafkainfra.NewKnowledgeDeletionDispatcher,
			fx.As(new(ports.KnowledgeDeletionDispatcher)),
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
			fx.As(new(ports.StoreRepository)),
			fx.As(new(ports.ProductCapacityRepository)),
			fx.As(new(ports.ActiveStoreLookupRepository)),
		),

		storegrpc.NewServer,

		fx.Annotate(
			postgres.NewKnowledgeRepository,
			fx.As(new(ports.KnowledgeRepository)),
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
