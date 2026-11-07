package dto

import "github.com/google/uuid"

// RecommendStoresInput contains the optional sources for Store recommendations.
type RecommendStoresInput struct {
	Query   *string
	StoreID *uuid.UUID
}

// RecommendStoresOutput contains the final reranked Store recommendations.
type RecommendStoresOutput struct {
	Recommendations []StoreRecommendation
}
