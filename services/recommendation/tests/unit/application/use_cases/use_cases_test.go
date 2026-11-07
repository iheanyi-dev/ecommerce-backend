package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/use_cases"
)

type fakeLogger struct{}

func (fakeLogger) Log(context.Context, ports.LogEvent) error {
	return nil
}

type fakeMetrics struct{}

func (fakeMetrics) Increment(context.Context, ports.Metric) error {
	return nil
}

func (fakeMetrics) Observe(context.Context, ports.Metric) error {
	return nil
}

type fakeSimilarityRetriever struct {
	candidates []ports.RecommendationCandidate
	err        error
	topK       int
	offset     int
}

func (f *fakeSimilarityRetriever) FindSimilarProducts(
	_ context.Context,
	_ string,
	topK int,
	offset int,
) ([]ports.RecommendationCandidate, error) {
	f.topK = topK
	f.offset = offset
	return f.candidates, f.err
}

func (f *fakeSimilarityRetriever) FindSimilarStores(
	_ context.Context,
	_ string,
	topK int,
	offset int,
) ([]ports.RecommendationCandidate, error) {
	f.topK = topK
	f.offset = offset
	return f.candidates, f.err
}

type fakeProductBehavioralModel struct {
	candidates []ports.RecommendationCandidate
	err        error
	limit      int
}

func (f *fakeProductBehavioralModel) FindAssociated(
	_ context.Context,
	_ uuid.UUID,
	limit int,
) ([]ports.RecommendationCandidate, error) {
	f.limit = limit
	return f.candidates, f.err
}

type fakeStoreBehavioralModel struct {
	candidates []ports.RecommendationCandidate
	err        error
	limit      int
}

func (f *fakeStoreBehavioralModel) FindAssociated(
	_ context.Context,
	_ uuid.UUID,
	limit int,
) ([]ports.RecommendationCandidate, error) {
	f.limit = limit
	return f.candidates, f.err
}

type fakeReranker struct {
	products               []dto.ProductRecommendation
	stores                 []dto.StoreRecommendation
	similarProductCount    int
	associatedProductCount int
	similarStoreCount      int
	associatedStoreCount   int
	err                    error
}

func (f *fakeReranker) RerankProducts(
	_ context.Context,
	similar []dto.ProductRecommendation,
	associated []dto.ProductRecommendation,
) ([]dto.ProductRecommendation, error) {
	f.similarProductCount = len(similar)
	f.associatedProductCount = len(associated)
	return f.products, f.err
}

func (f *fakeReranker) RerankStores(
	_ context.Context,
	similar []dto.StoreRecommendation,
	associated []dto.StoreRecommendation,
) ([]dto.StoreRecommendation, error) {
	f.similarStoreCount = len(similar)
	f.associatedStoreCount = len(associated)
	return f.stores, f.err
}

func TestFindSimilarProducts(t *testing.T) {
	productID := uuid.New()
	retriever := &fakeSimilarityRetriever{
		candidates: []ports.RecommendationCandidate{
			{ID: productID, Score: 0.95},
		},
	}

	uc := use_cases.FindSimilarProducts{
		SimilarityRetriever: retriever,
		Logger:              fakeLogger{},
		Metrics:             fakeMetrics{},
	}

	result, err := uc.Execute(context.Background(), dto.FindSimilarProductsInput{
		Query: "laptop",
		Page:  3,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(result.Recommendations))
	}
	if result.Recommendations[0].ProductID != productID {
		t.Fatal("unexpected Product ID")
	}
	if result.Recommendations[0].Score != 0.95 {
		t.Fatal("unexpected recommendation score")
	}
	if retriever.topK != 10 || retriever.offset != 20 {
		t.Fatalf("unexpected retrieval pagination: topK=%d offset=%d", retriever.topK, retriever.offset)
	}
}

func TestFindSimilarProductsRejectsInvalidPage(t *testing.T) {
	uc := use_cases.FindSimilarProducts{
		SimilarityRetriever: &fakeSimilarityRetriever{},
		Logger:              fakeLogger{},
		Metrics:             fakeMetrics{},
	}

	_, err := uc.Execute(context.Background(), dto.FindSimilarProductsInput{Page: 0})
	if err == nil {
		t.Fatal("expected an error for an invalid page")
	}
}

func TestFindSimilarProductsPropagatesRetrieverError(t *testing.T) {
	expectedErr := errors.New("retrieval failed")
	uc := use_cases.FindSimilarProducts{
		SimilarityRetriever: &fakeSimilarityRetriever{err: expectedErr},
		Logger:              fakeLogger{},
		Metrics:             fakeMetrics{},
	}

	_, err := uc.Execute(context.Background(), dto.FindSimilarProductsInput{
		Query: "laptop",
		Page:  1,
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected retrieval error, got %v", err)
	}
}

func TestFindSimilarStores(t *testing.T) {
	storeID := uuid.New()
	retriever := &fakeSimilarityRetriever{
		candidates: []ports.RecommendationCandidate{
			{ID: storeID, Score: 0.87},
		},
	}

	uc := use_cases.FindSimilarStores{
		SimilarityRetriever: retriever,
		Logger:              fakeLogger{},
		Metrics:             fakeMetrics{},
	}

	result, err := uc.Execute(context.Background(), dto.FindSimilarStoresInput{
		Query: "electronics",
		Page:  2,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(result.Recommendations))
	}
	if result.Recommendations[0].StoreID != storeID {
		t.Fatal("unexpected Store ID")
	}
	if retriever.topK != 10 || retriever.offset != 10 {
		t.Fatalf("unexpected retrieval pagination: topK=%d offset=%d", retriever.topK, retriever.offset)
	}
}

func TestFindAssociatedProducts(t *testing.T) {
	productID := uuid.New()
	candidates := make([]ports.RecommendationCandidate, 15)

	for i := range candidates {
		candidates[i] = ports.RecommendationCandidate{
			ID:    uuid.New(),
			Score: float64(i) / 15,
		}
	}

	model := &fakeProductBehavioralModel{candidates: candidates}
	uc := use_cases.FindAssociatedProducts{
		BehavioralModel: model,
		Logger:          fakeLogger{},
		Metrics:         fakeMetrics{},
	}

	result, err := uc.Execute(context.Background(), dto.FindAssociatedProductsInput{
		ProductID: productID,
		Page:      2,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if model.limit != 20 {
		t.Fatalf("expected model limit 20, got %d", model.limit)
	}
	if len(result.Recommendations) != 5 {
		t.Fatalf("expected 5 recommendations, got %d", len(result.Recommendations))
	}
	if result.Recommendations[0].ProductID != candidates[10].ID {
		t.Fatal("unexpected first recommendation on page 2")
	}
}

func TestFindAssociatedProductsRejectsInvalidPage(t *testing.T) {
	uc := use_cases.FindAssociatedProducts{
		BehavioralModel: &fakeProductBehavioralModel{},
		Logger:          fakeLogger{},
		Metrics:         fakeMetrics{},
	}

	_, err := uc.Execute(context.Background(), dto.FindAssociatedProductsInput{
		ProductID: uuid.New(),
		Page:      0,
	})
	if err == nil {
		t.Fatal("expected an error for an invalid page")
	}
}

func TestFindAssociatedStores(t *testing.T) {
	storeID := uuid.New()
	candidates := make([]ports.RecommendationCandidate, 12)

	for i := range candidates {
		candidates[i] = ports.RecommendationCandidate{
			ID:    uuid.New(),
			Score: float64(i) / 12,
		}
	}

	model := &fakeStoreBehavioralModel{candidates: candidates}
	uc := use_cases.FindAssociatedStores{
		BehavioralModel: model,
		Logger:          fakeLogger{},
		Metrics:         fakeMetrics{},
	}

	result, err := uc.Execute(context.Background(), dto.FindAssociatedStoresInput{
		StoreID: storeID,
		Page:    1,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if model.limit != 10 {
		t.Fatalf("expected model limit 10, got %d", model.limit)
	}
	if len(result.Recommendations) != 10 {
		t.Fatalf("expected 10 recommendations, got %d", len(result.Recommendations))
	}
	if result.Recommendations[0].StoreID != candidates[0].ID {
		t.Fatal("unexpected first recommendation")
	}
}

func TestRecommendProducts(t *testing.T) {
	similarID := uuid.New()
	associatedID := uuid.New()
	finalID := uuid.New()

	similarRetriever := &fakeSimilarityRetriever{
		candidates: []ports.RecommendationCandidate{
			{ID: similarID, Score: 0.9},
		},
	}
	associatedModel := &fakeProductBehavioralModel{
		candidates: []ports.RecommendationCandidate{
			{ID: associatedID, Score: 0.8},
		},
	}
	reranker := &fakeReranker{
		products: []dto.ProductRecommendation{
			{ProductID: finalID, Score: 0.98},
		},
	}

	uc := use_cases.RecommendProducts{
		FindSimilarProducts: &use_cases.FindSimilarProducts{
			SimilarityRetriever: similarRetriever,
			Logger:              fakeLogger{},
			Metrics:             fakeMetrics{},
		},
		FindAssociatedProducts: &use_cases.FindAssociatedProducts{
			BehavioralModel: associatedModel,
			Logger:          fakeLogger{},
			Metrics:         fakeMetrics{},
		},
		Reranker: reranker,
		Logger:   fakeLogger{},
		Metrics:  fakeMetrics{},
	}

	query := "laptop"
	productID := uuid.New()

	result, err := uc.Execute(context.Background(), dto.RecommendProductsInput{
		Query:     &query,
		ProductID: &productID,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if reranker.similarProductCount != 1 || reranker.associatedProductCount != 1 {
		t.Fatal("expected both candidate sources to reach the reranker")
	}
	if len(result.Recommendations) != 1 || result.Recommendations[0].ProductID != finalID {
		t.Fatal("unexpected final Product recommendations")
	}
}

func TestRecommendStores(t *testing.T) {
	similarID := uuid.New()
	associatedID := uuid.New()
	finalID := uuid.New()

	similarRetriever := &fakeSimilarityRetriever{
		candidates: []ports.RecommendationCandidate{
			{ID: similarID, Score: 0.9},
		},
	}
	associatedModel := &fakeStoreBehavioralModel{
		candidates: []ports.RecommendationCandidate{
			{ID: associatedID, Score: 0.8},
		},
	}
	reranker := &fakeReranker{
		stores: []dto.StoreRecommendation{
			{StoreID: finalID, Score: 0.98},
		},
	}

	uc := use_cases.RecommendStores{
		FindSimilarStores: &use_cases.FindSimilarStores{
			SimilarityRetriever: similarRetriever,
			Logger:              fakeLogger{},
			Metrics:             fakeMetrics{},
		},
		FindAssociatedStores: &use_cases.FindAssociatedStores{
			BehavioralModel: associatedModel,
			Logger:          fakeLogger{},
			Metrics:         fakeMetrics{},
		},
		Reranker: reranker,
		Logger:   fakeLogger{},
		Metrics:  fakeMetrics{},
	}

	query := "electronics"
	storeID := uuid.New()

	result, err := uc.Execute(context.Background(), dto.RecommendStoresInput{
		Query:   &query,
		StoreID: &storeID,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if reranker.similarStoreCount != 1 || reranker.associatedStoreCount != 1 {
		t.Fatal("expected both candidate sources to reach the reranker")
	}
	if len(result.Recommendations) != 1 || result.Recommendations[0].StoreID != finalID {
		t.Fatal("unexpected final Store recommendations")
	}
}
