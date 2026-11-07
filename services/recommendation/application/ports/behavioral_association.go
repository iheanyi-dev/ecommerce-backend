package ports

import (
	"context"

	"github.com/google/uuid"
)

// ProductBehavioralAssociationModel performs Product behavioral inference.
type ProductBehavioralAssociationModel interface {
	FindAssociated(
		ctx context.Context,
		productID uuid.UUID,
		limit int,
	) ([]RecommendationCandidate, error)
}

// StoreBehavioralAssociationModel performs Store behavioral inference.
type StoreBehavioralAssociationModel interface {
	FindAssociated(
		ctx context.Context,
		storeID uuid.UUID,
		limit int,
	) ([]RecommendationCandidate, error)
}
