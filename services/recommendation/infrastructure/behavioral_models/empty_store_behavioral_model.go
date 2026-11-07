package behavioral_models

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// EmptyStoreBehavioralAssociationModel is a temporary Store behavioral
// model used until a trained model is available.
type EmptyStoreBehavioralAssociationModel struct{}

// FindAssociated returns no behavioral associations.
func (m *EmptyStoreBehavioralAssociationModel) FindAssociated(
	_ context.Context,
	_ uuid.UUID,
	_ int,
) ([]ports.RecommendationCandidate, error) {
	return []ports.RecommendationCandidate{}, nil
}
