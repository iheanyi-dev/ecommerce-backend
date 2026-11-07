package schemas_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/schemas"
)

func TestNewStoreRecommendationResponse(t *testing.T) {
	storeID := uuid.New()

	response := schemas.NewStoreRecommendationResponse([]dto.StoreRecommendation{
		{StoreID: storeID, Score: 0.95},
	})

	if len(response.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(response.Recommendations))
	}

	if response.Recommendations[0].EntityID != storeID {
		t.Fatalf("expected store ID %s, got %s", storeID, response.Recommendations[0].EntityID)
	}

	if response.Recommendations[0].Score != 0.95 {
		t.Fatalf("expected score 0.95, got %f", response.Recommendations[0].Score)
	}
}

func TestNewProductRecommendationResponse(t *testing.T) {
	productID := uuid.New()

	response := schemas.NewProductRecommendationResponse([]dto.ProductRecommendation{
		{ProductID: productID, Score: 0.85},
	})

	if len(response.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(response.Recommendations))
	}

	if response.Recommendations[0].EntityID != productID {
		t.Fatalf("expected product ID %s, got %s", productID, response.Recommendations[0].EntityID)
	}

	if response.Recommendations[0].Score != 0.85 {
		t.Fatalf("expected score 0.85, got %f", response.Recommendations[0].Score)
	}
}
