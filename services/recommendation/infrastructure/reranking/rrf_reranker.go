package reranking

import (
	"context"
	"sort"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

const rrfRerankingK = 60

// RRFReranker fuses semantic and behavioral recommendation rankings.
type RRFReranker struct{}

// NewRRFReranker creates a reciprocal rank fusion reranker.
func NewRRFReranker() *RRFReranker {
	return &RRFReranker{}
}

// RerankStores fuses semantic and behavioral Store rankings.
func (r *RRFReranker) RerankStores(
	_ context.Context,
	similarStores []dto.StoreRecommendation,
	associatedStores []dto.StoreRecommendation,
) ([]dto.StoreRecommendation, error) {
	return rerank(
		similarStores,
		associatedStores,
		func(recommendation dto.StoreRecommendation) interface{} {
			return recommendation.StoreID
		},
	), nil
}

// RerankProducts fuses semantic and behavioral Product rankings.
func (r *RRFReranker) RerankProducts(
	_ context.Context,
	similarProducts []dto.ProductRecommendation,
	associatedProducts []dto.ProductRecommendation,
) ([]dto.ProductRecommendation, error) {
	return rerank(
		similarProducts,
		associatedProducts,
		func(recommendation dto.ProductRecommendation) interface{} {
			return recommendation.ProductID
		},
	), nil
}

type rankedCandidate[T any] struct {
	recommendation T
	fusedScore     float64
	firstSeen      int
}

// rerank fuses two ranked candidate lists using reciprocal rank fusion.
func rerank[T any](
	primary []T,
	secondary []T,
	key func(T) interface{},
) []T {
	candidates := make(map[interface{}]*rankedCandidate[T])
	sequence := 0

	add := func(recommendations []T) {
		for rank, recommendation := range recommendations {
			candidateID := key(recommendation)

			candidate, exists := candidates[candidateID]
			if !exists {
				candidate = &rankedCandidate[T]{
					recommendation: recommendation,
					firstSeen:      sequence,
				}
				candidates[candidateID] = candidate
				sequence++
			}

			candidate.fusedScore += 1.0 / float64(rrfRerankingK+rank+1)
		}
	}

	add(primary)
	add(secondary)

	ordered := make([]*rankedCandidate[T], 0, len(candidates))
	for _, candidate := range candidates {
		ordered = append(ordered, candidate)
	}

	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].fusedScore != ordered[j].fusedScore {
			return ordered[i].fusedScore > ordered[j].fusedScore
		}
		return ordered[i].firstSeen < ordered[j].firstSeen
	})

	result := make([]T, 0, len(ordered))
	for _, candidate := range ordered {
		result = append(result, candidate.recommendation)
	}

	return result
}

var _ ports.Reranker = (*RRFReranker)(nil)
