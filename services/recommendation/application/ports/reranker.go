package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
)

// Reranker combines and reranks recommendation candidates.
type Reranker interface {
	RerankProducts(
		ctx context.Context,
		similarProducts []dto.ProductRecommendation,
		associatedProducts []dto.ProductRecommendation,
	) ([]dto.ProductRecommendation, error)

	RerankStores(
		ctx context.Context,
		similarStores []dto.StoreRecommendation,
		associatedStores []dto.StoreRecommendation,
	) ([]dto.StoreRecommendation, error)
}
