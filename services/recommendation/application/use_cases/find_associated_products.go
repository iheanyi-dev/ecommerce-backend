package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// FindAssociatedProducts retrieves Products associated by user behavior.
type FindAssociatedProducts struct {
	BehavioralModel ports.ProductBehavioralAssociationModel
	Logger          ports.Logger
	Metrics         ports.Metrics
}

// Execute retrieves the requested page of associated Products.
func (uc *FindAssociatedProducts) Execute(
	ctx context.Context,
	input dto.FindAssociatedProductsInput,
) (dto.FindAssociatedProductsOutput, error) {
	if input.Page < 1 {
		return dto.FindAssociatedProductsOutput{}, fmt.Errorf("page must be greater than or equal to 1")
	}

	candidates, err := uc.BehavioralModel.FindAssociated(
		ctx,
		input.ProductID,
		input.Page*pageSize,
	)
	if err != nil {
		return dto.FindAssociatedProductsOutput{}, fmt.Errorf("find associated products: %w", err)
	}

	offset := (input.Page - 1) * pageSize
	recommendations := make([]dto.ProductRecommendation, 0)
	if offset < len(candidates) {
		for _, candidate := range candidates[offset:] {
			recommendations = append(recommendations, dto.ProductRecommendation{
				ProductID: candidate.ID,
				Score:     candidate.Score,
			})
		}
	}

	_ = uc.Metrics.Increment(ctx, ports.Metric{
		Name:  "recommendation.product_associations.queries",
		Value: 1,
	})
	_ = uc.Logger.Log(ctx, ports.LogEvent{
		Event:     "recommendation.product_associations.found",
		Operation: "find_associated_products",
	})

	return dto.FindAssociatedProductsOutput{Recommendations: recommendations}, nil
}
