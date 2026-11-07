package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// FindSimilarStores retrieves semantic Store recommendations.
type FindSimilarStores struct {
	SimilarityRetriever ports.SimilarityRetriever
	Logger              ports.Logger
	Metrics             ports.Metrics
}

// Execute retrieves the requested page of similar Stores.
func (uc *FindSimilarStores) Execute(
	ctx context.Context,
	input dto.FindSimilarStoresInput,
) (dto.FindSimilarStoresOutput, error) {
	if input.Page < 1 {
		return dto.FindSimilarStoresOutput{}, fmt.Errorf("page must be greater than or equal to 1")
	}

	candidates, err := uc.SimilarityRetriever.FindSimilarStores(
		ctx,
		input.Query,
		pageSize,
		(input.Page-1)*pageSize,
	)
	if err != nil {
		return dto.FindSimilarStoresOutput{}, fmt.Errorf("find similar stores: %w", err)
	}

	recommendations := make([]dto.StoreRecommendation, 0, len(candidates))
	for _, candidate := range candidates {
		recommendations = append(recommendations, dto.StoreRecommendation{
			StoreID: candidate.ID,
			Score:   candidate.Score,
		})
	}

	_ = uc.Metrics.Increment(ctx, ports.Metric{
		Name:  "recommendation.store_similarity.queries",
		Value: 1,
	})
	_ = uc.Logger.Log(ctx, ports.LogEvent{
		Event:     "recommendation.store_similarity.found",
		Operation: "find_similar_stores",
	})

	return dto.FindSimilarStoresOutput{Recommendations: recommendations}, nil
}
