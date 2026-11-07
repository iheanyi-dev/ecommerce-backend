package behavioral_models

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// EmptyProductBehavioralAssociationModel is a temporary Product behavioral
// model used until a trained model is available.
type EmptyProductBehavioralAssociationModel struct{}

// FindAssociated returns no behavioral associations.
func (m *EmptyProductBehavioralAssociationModel) FindAssociated(
	_ context.Context,
	_ uuid.UUID,
	_ int,
) ([]ports.RecommendationCandidate, error) {
	return []ports.RecommendationCandidate{}, nil
}
