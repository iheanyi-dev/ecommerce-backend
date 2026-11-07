package ports

import (
	"context"

	"github.com/google/uuid"
)

// SimilarityRetriever retrieves ranked candidates from the Embedding service.
type SimilarityRetriever interface {
	FindSimilarProducts(
		ctx context.Context,
		query string,
		topK int,
		offset int,
	) ([]RecommendationCandidate, error)

	FindSimilarStores(
		ctx context.Context,
		query string,
		topK int,
		offset int,
	) ([]RecommendationCandidate, error)
}

// RecommendationCandidate represents a candidate ID and its similarity score.
type RecommendationCandidate struct {
	ID    uuid.UUID
	Score float64
}
