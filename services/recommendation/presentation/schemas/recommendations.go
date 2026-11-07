package schemas

import (
	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
)

// RecommendationItem represents a recommended entity and its score.
type RecommendationItem struct {
	EntityID uuid.UUID `json:"entity_id"`
	Score    float64   `json:"score"`
}

// RecommendationResponse contains the final recommendations.
type RecommendationResponse struct {
	Recommendations []RecommendationItem `json:"recommendations"`
}

// NewStoreRecommendationResponse maps Store recommendations to HTTP output.
func NewStoreRecommendationResponse(
	recommendations []dto.StoreRecommendation,
) RecommendationResponse {
	items := make([]RecommendationItem, 0, len(recommendations))

	for _, recommendation := range recommendations {
		items = append(items, RecommendationItem{
			EntityID: recommendation.StoreID,
			Score:    recommendation.Score,
		})
	}

	return RecommendationResponse{Recommendations: items}
}

// NewProductRecommendationResponse maps Product recommendations to HTTP output.
func NewProductRecommendationResponse(
	recommendations []dto.ProductRecommendation,
) RecommendationResponse {
	items := make([]RecommendationItem, 0, len(recommendations))

	for _, recommendation := range recommendations {
		items = append(items, RecommendationItem{
			EntityID: recommendation.ProductID,
			Score:    recommendation.Score,
		})
	}

	return RecommendationResponse{Recommendations: items}
}
