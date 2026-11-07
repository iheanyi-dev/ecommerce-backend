package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

const pageSize = 10

// FindSimilarProducts retrieves semantic Product recommendations.
type FindSimilarProducts struct {
	SimilarityRetriever ports.SimilarityRetriever
	Logger              ports.Logger
	Metrics             ports.Metrics
}

// Execute retrieves the requested page of similar Products.
func (uc *FindSimilarProducts) Execute(
	ctx context.Context,
	input dto.FindSimilarProductsInput,
) (dto.FindSimilarProductsOutput, error) {
	if input.Page < 1 {
		return dto.FindSimilarProductsOutput{}, fmt.Errorf("page must be greater than or equal to 1")
	}

	candidates, err := uc.SimilarityRetriever.FindSimilarProducts(
		ctx,
		input.Query,
		pageSize,
		(input.Page-1)*pageSize,
	)
	if err != nil {
		return dto.FindSimilarProductsOutput{}, fmt.Errorf("find similar products: %w", err)
	}

	recommendations := make([]dto.ProductRecommendation, 0, len(candidates))
	for _, candidate := range candidates {
		recommendations = append(recommendations, dto.ProductRecommendation{
			ProductID: candidate.ID,
			Score:     candidate.Score,
		})
	}

	_ = uc.Metrics.Increment(ctx, ports.Metric{
		Name:  "recommendation.product_similarity.queries",
		Value: 1,
	})
	_ = uc.Logger.Log(ctx, ports.LogEvent{
		Event:     "recommendation.product_similarity.found",
		Operation: "find_similar_products",
	})

	return dto.FindSimilarProductsOutput{Recommendations: recommendations}, nil
}
