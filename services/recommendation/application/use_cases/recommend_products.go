package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// RecommendProducts orchestrates Product candidate generation and reranking.
type RecommendProducts struct {
	FindSimilarProducts    *FindSimilarProducts
	FindAssociatedProducts *FindAssociatedProducts
	Reranker               ports.Reranker
	Logger                 ports.Logger
	Metrics                ports.Metrics
}

// Execute generates and reranks Product recommendations.
func (uc *RecommendProducts) Execute(
	ctx context.Context,
	input dto.RecommendProductsInput,
) (dto.RecommendProductsOutput, error) {
	var similarProducts []dto.ProductRecommendation
	if input.Query != nil {
		result, err := uc.FindSimilarProducts.Execute(ctx, dto.FindSimilarProductsInput{
			Query: *input.Query,
			Page:  1,
		})
		if err != nil {
			return dto.RecommendProductsOutput{}, fmt.Errorf("retrieve similar products: %w", err)
		}
		similarProducts = result.Recommendations
	}

	var associatedProducts []dto.ProductRecommendation
	if input.ProductID != nil {
		result, err := uc.FindAssociatedProducts.Execute(ctx, dto.FindAssociatedProductsInput{
			ProductID: *input.ProductID,
			Page:      1,
		})
		if err != nil {
			return dto.RecommendProductsOutput{}, fmt.Errorf("retrieve associated products: %w", err)
		}
		associatedProducts = result.Recommendations
	}

	recommendations, err := uc.Reranker.RerankProducts(
		ctx,
		similarProducts,
		associatedProducts,
	)
	if err != nil {
		return dto.RecommendProductsOutput{}, fmt.Errorf("rerank products: %w", err)
	}

	_ = uc.Metrics.Increment(ctx, ports.Metric{
		Name:  "recommendation.product.queries",
		Value: 1,
	})
	_ = uc.Logger.Log(ctx, ports.LogEvent{
		Event:     "recommendation.products.completed",
		Operation: "recommend_products",
	})

	return dto.RecommendProductsOutput{Recommendations: recommendations}, nil
}
