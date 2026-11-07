package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// FindAssociatedStores retrieves Stores associated by user behavior.
type FindAssociatedStores struct {
	BehavioralModel ports.StoreBehavioralAssociationModel
	Logger          ports.Logger
	Metrics         ports.Metrics
}

// Execute retrieves the requested page of associated Stores.
func (uc *FindAssociatedStores) Execute(
	ctx context.Context,
	input dto.FindAssociatedStoresInput,
) (dto.FindAssociatedStoresOutput, error) {
	if input.Page < 1 {
		return dto.FindAssociatedStoresOutput{}, fmt.Errorf("page must be greater than or equal to 1")
	}

	candidates, err := uc.BehavioralModel.FindAssociated(
		ctx,
		input.StoreID,
		input.Page*pageSize,
	)
	if err != nil {
		return dto.FindAssociatedStoresOutput{}, fmt.Errorf("find associated stores: %w", err)
	}

	offset := (input.Page - 1) * pageSize
	recommendations := make([]dto.StoreRecommendation, 0)
	if offset < len(candidates) {
		end := min(offset+pageSize, len(candidates))
		for _, candidate := range candidates[offset:end] {
			recommendations = append(recommendations, dto.StoreRecommendation{
				StoreID: candidate.ID,
				Score:   candidate.Score,
			})
		}
	}

	_ = uc.Metrics.Increment(ctx, ports.Metric{
		Name:  "recommendation.store_associations.queries",
		Value: 1,
	})
	_ = uc.Logger.Log(ctx, ports.LogEvent{
		Event:     "recommendation.store_associations.found",
		Operation: "find_associated_stores",
	})

	return dto.FindAssociatedStoresOutput{Recommendations: recommendations}, nil
}
