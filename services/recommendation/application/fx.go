package application

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/use_cases"
)

func newFindSimilarStores(
	retriever ports.SimilarityRetriever,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.FindSimilarStores {
	return &use_cases.FindSimilarStores{
		SimilarityRetriever: retriever,
		Logger:              logger,
		Metrics:             metrics,
	}
}

func newFindSimilarProducts(
	retriever ports.SimilarityRetriever,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.FindSimilarProducts {
	return &use_cases.FindSimilarProducts{
		SimilarityRetriever: retriever,
		Logger:              logger,
		Metrics:             metrics,
	}
}

func newFindAssociatedStores(
	model ports.StoreBehavioralAssociationModel,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.FindAssociatedStores {
	return &use_cases.FindAssociatedStores{
		BehavioralModel: model,
		Logger:          logger,
		Metrics:         metrics,
	}
}

func newFindAssociatedProducts(
	model ports.ProductBehavioralAssociationModel,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.FindAssociatedProducts {
	return &use_cases.FindAssociatedProducts{
		BehavioralModel: model,
		Logger:          logger,
		Metrics:         metrics,
	}
}

func newRecommendStores(
	similar *use_cases.FindSimilarStores,
	associated *use_cases.FindAssociatedStores,
	reranker ports.Reranker,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.RecommendStores {
	return &use_cases.RecommendStores{
		FindSimilarStores:    similar,
		FindAssociatedStores: associated,
		Reranker:             reranker,
		Logger:               logger,
		Metrics:              metrics,
	}
}

func newRecommendProducts(
	similar *use_cases.FindSimilarProducts,
	associated *use_cases.FindAssociatedProducts,
	reranker ports.Reranker,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.RecommendProducts {
	return &use_cases.RecommendProducts{
		FindSimilarProducts:    similar,
		FindAssociatedProducts: associated,
		Reranker:               reranker,
		Logger:                 logger,
		Metrics:                metrics,
	}
}

var Module = fx.Module(
	"recommendation-application",
	fx.Provide(
		newFindSimilarStores,
		newFindSimilarProducts,
		newFindAssociatedStores,
		newFindAssociatedProducts,
		newRecommendStores,
		newRecommendProducts,
	),
)
