package dto

import "github.com/google/uuid"

// FindSimilarStoresInput contains the query and page for semantic Store search.
type FindSimilarStoresInput struct {
	Query string
	Page  int
}

// StoreRecommendation represents a Store candidate and its similarity score.
type StoreRecommendation struct {
	StoreID uuid.UUID
	Score   float64
}

// FindSimilarStoresOutput contains the requested page of Store recommendations.
type FindSimilarStoresOutput struct {
	Recommendations []StoreRecommendation
}
