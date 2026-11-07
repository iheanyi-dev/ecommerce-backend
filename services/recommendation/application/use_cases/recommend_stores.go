package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// RecommendStores orchestrates Store candidate generation and reranking.
type RecommendStores struct {
	FindSimilarStores    *FindSimilarStores
	FindAssociatedStores *FindAssociatedStores
	Reranker             ports.Reranker
	Logger               ports.Logger
	Metrics              ports.Metrics
}

// Execute generates and reranks Store recommendations.
func (uc *RecommendStores) Execute(
	ctx context.Context,
	input dto.RecommendStoresInput,
) (dto.RecommendStoresOutput, error) {
	var similarStores []dto.StoreRecommendation
	if input.Query != nil {
		result, err := uc.FindSimilarStores.Execute(ctx, dto.FindSimilarStoresInput{
			Query: *input.Query,
			Page:  1,
		})
		if err != nil {
			return dto.RecommendStoresOutput{}, fmt.Errorf("retrieve similar stores: %w", err)
		}
		similarStores = result.Recommendations
	}

	var associatedStores []dto.StoreRecommendation
	if input.StoreID != nil {
		result, err := uc.FindAssociatedStores.Execute(ctx, dto.FindAssociatedStoresInput{
			StoreID: *input.StoreID,
			Page:    1,
		})
		if err != nil {
			return dto.RecommendStoresOutput{}, fmt.Errorf("retrieve associated stores: %w", err)
		}
		associatedStores = result.Recommendations
	}

	recommendations, err := uc.Reranker.RerankStores(
		ctx,
		similarStores,
		associatedStores,
	)
	if err != nil {
		return dto.RecommendStoresOutput{}, fmt.Errorf("rerank stores: %w", err)
	}

	_ = uc.Metrics.Increment(ctx, ports.Metric{
		Name:  "recommendation.store.queries",
		Value: 1,
	})
	_ = uc.Logger.Log(ctx, ports.LogEvent{
		Event:     "recommendation.stores.completed",
		Operation: "recommend_stores",
	})

	return dto.RecommendStoresOutput{Recommendations: recommendations}, nil
}
