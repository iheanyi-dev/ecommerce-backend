package dto

import "github.com/google/uuid"

// FindSimilarProductsInput contains the query and page for semantic Product search.
type FindSimilarProductsInput struct {
	Query string
	Page  int
}

// ProductRecommendation represents a Product candidate and its similarity score.
type ProductRecommendation struct {
	ProductID uuid.UUID
	Score     float64
}

// FindSimilarProductsOutput contains the requested page of Product recommendations.
type FindSimilarProductsOutput struct {
	Recommendations []ProductRecommendation
}
