package reranking_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/reranking"
)

func TestRerankProducts(t *testing.T) {
	reranker := reranking.NewRRFReranker()

	productA := uuid.New()
	productB := uuid.New()
	productC := uuid.New()

	tests := []struct {
		name      string
		primary   []dto.ProductRecommendation
		secondary []dto.ProductRecommendation
		expected  []uuid.UUID
	}{
		{
			name: "fuses duplicate candidates across rankings",
			primary: []dto.ProductRecommendation{
				{ProductID: productA, Score: 0.91},
				{ProductID: productB, Score: 0.82},
			},
			secondary: []dto.ProductRecommendation{
				{ProductID: productB, Score: 0.73},
				{ProductID: productC, Score: 0.64},
			},
			expected: []uuid.UUID{productB, productA, productC},
		},
		{
			name: "preserves first-seen order for equal scores",
			primary: []dto.ProductRecommendation{
				{ProductID: productA, Score: 0.91},
			},
			secondary: []dto.ProductRecommendation{
				{ProductID: productB, Score: 0.82},
			},
			expected: []uuid.UUID{productA, productB},
		},
		{
			name:     "handles empty rankings",
			expected: []uuid.UUID{},
		},
		{
			name: "handles empty primary ranking",
			secondary: []dto.ProductRecommendation{
				{ProductID: productC, Score: 0.64},
			},
			expected: []uuid.UUID{productC},
		},
		{
			name: "handles empty secondary ranking",
			primary: []dto.ProductRecommendation{
				{ProductID: productA, Score: 0.91},
			},
			expected: []uuid.UUID{productA},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reranker.RerankProducts(
				context.Background(),
				tt.primary,
				tt.secondary,
			)
			if err != nil {
				t.Fatalf("rerank products: %v", err)
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d recommendations, got %d", len(tt.expected), len(got))
			}

			for i, expectedID := range tt.expected {
				if got[i].ProductID != expectedID {
					t.Errorf("position %d: expected product %s, got %s",
						i, expectedID, got[i].ProductID)
				}
			}
		})
	}
}

func TestRerankProductsPreservesOriginalCandidateScores(t *testing.T) {
	reranker := reranking.NewRRFReranker()

	productA := uuid.New()
	productB := uuid.New()

	primary := []dto.ProductRecommendation{
		{ProductID: productA, Score: 0.91},
		{ProductID: productB, Score: 0.82},
	}
	secondary := []dto.ProductRecommendation{
		{ProductID: productB, Score: 0.73},
	}

	got, err := reranker.RerankProducts(context.Background(), primary, secondary)
	if err != nil {
		t.Fatalf("rerank products: %v", err)
	}

	if got[0].ProductID != productB {
		t.Fatalf("expected product B first, got %s", got[0].ProductID)
	}
	if got[0].Score != 0.82 {
		t.Errorf("expected original primary score 0.82, got %f", got[0].Score)
	}
	if got[1].Score != 0.91 {
		t.Errorf("expected original score 0.91, got %f", got[1].Score)
	}
}

func TestRerankStores(t *testing.T) {
	reranker := reranking.NewRRFReranker()

	storeA := uuid.New()
	storeB := uuid.New()
	storeC := uuid.New()

	primary := []dto.StoreRecommendation{
		{StoreID: storeA, Score: 0.91},
		{StoreID: storeB, Score: 0.82},
	}
	secondary := []dto.StoreRecommendation{
		{StoreID: storeB, Score: 0.73},
		{StoreID: storeC, Score: 0.64},
	}

	got, err := reranker.RerankStores(context.Background(), primary, secondary)
	if err != nil {
		t.Fatalf("rerank stores: %v", err)
	}

	expected := []uuid.UUID{storeB, storeA, storeC}
	if len(got) != len(expected) {
		t.Fatalf("expected %d recommendations, got %d", len(expected), len(got))
	}

	for i, expectedID := range expected {
		if got[i].StoreID != expectedID {
			t.Errorf("position %d: expected store %s, got %s",
				i, expectedID, got[i].StoreID)
		}
	}

	if got[0].Score != 0.82 {
		t.Errorf("expected original primary score 0.82, got %f", got[0].Score)
	}
}
