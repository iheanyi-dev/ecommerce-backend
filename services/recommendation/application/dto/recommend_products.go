package dto

import "github.com/google/uuid"

// RecommendProductsInput contains the optional sources for Product recommendations.
type RecommendProductsInput struct {
	Query     *string
	ProductID *uuid.UUID
}

// RecommendProductsOutput contains the final reranked Product recommendations.
type RecommendProductsOutput struct {
	Recommendations []ProductRecommendation
}
