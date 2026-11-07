package infrastructure

import (
	"context"
	"os"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/behavioral_models"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/embedding"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/reranking"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/shared/config"
)

func newSimilarityRetriever(cfg *config.Config) (*embedding.GRPCClient, error) {
	return embedding.NewGRPCClient(
		cfg.EmbeddingServiceAddr,
		cfg.EmbeddingServiceName,
		cfg.EmbeddingServiceSecret,
		cfg.EmbeddingGRPCTimeout,
	)
}

func newLogger() ports.Logger {
	return observability.NewStructuredLogger(os.Stdout)
}

func newMetrics() ports.Metrics {
	return observability.NewMetrics()
}

func newTracer(cfg *config.Config) (ports.Tracer, error) {
	return observability.NewOpenTelemetryTracer(cfg.ServiceName)
}

func newReranker() ports.Reranker {
	return reranking.NewRRFReranker()
}

func newStoreBehavioralModel() ports.StoreBehavioralAssociationModel {
	return &behavioral_models.EmptyStoreBehavioralAssociationModel{}
}

func newProductBehavioralModel() ports.ProductBehavioralAssociationModel {
	return &behavioral_models.EmptyProductBehavioralAssociationModel{}
}

var Module = fx.Module(
	"recommendation-infrastructure",
	fx.Provide(
		newSimilarityRetriever,
		fx.Annotate(
			newSimilarityRetriever,
			fx.As(new(ports.SimilarityRetriever)),
		),
		newLogger,
		newMetrics,
		newTracer,
		newReranker,
		newStoreBehavioralModel,
		newProductBehavioralModel,
	),
	fx.Invoke(registerGRPCClientShutdown),
)

func registerGRPCClientShutdown(lc fx.Lifecycle, client *embedding.GRPCClient) {
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return client.Close()
		},
	})
}
