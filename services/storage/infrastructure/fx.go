package infrastructure

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/grpc"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/messaging"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/persistence/filesystem"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/shared/config"
	"net/http"

	"go.uber.org/fx"
)

// Module provides concrete Storage infrastructure implementations.
//
// Infrastructure implements application ports while keeping the application
// layer independent from filesystem, gRPC, and observability details.
var Module = fx.Module(
	"infrastructure",

	fx.Provide(
		func() *http.Client { return http.DefaultClient },
		fx.Annotate(
			func(cfg *config.Config) (*filesystem.ObjectStorage, error) {
				return filesystem.NewObjectStorage(cfg.StorageRoot)
			},
			fx.As(new(ports.ObjectStorage)),
		),

		fx.Annotate(
			observability.NewProductionLogger,
			fx.As(new(ports.Logger)),
		),

		fx.Annotate(
			observability.NewMetrics,
			fx.As(new(ports.Metrics)),
		),

		func() (ports.Tracer, error) {
			return observability.NewOpenTelemetryTracer("storage")
		},

		// Internal Storage gRPC adapter.
		grpc.NewServer,

		// Internal Kafka consumers for asynchronous object workflows.
		messaging.NewConsumer,
	),
)
