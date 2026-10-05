"""Tests for the Recommendation RRF reranker."""

from __future__ import annotations

from uuid import uuid4

import pytest

from application.dto.find_similar_products import ProductRecommendation
from application.dto.find_similar_stores import StoreRecommendation
from infrastructure.reranking.rrf_reranker import RRF_RERANKING_K, RRFReranker


def test_rerank_stores_combines_candidates_and_deduplicates_by_store_id():
    """Candidates appearing in both sources receive a combined RRF score."""
    reranker = RRFReranker()

    shared_id = uuid4()
    similar_only_id = uuid4()
    associated_only_id = uuid4()

    similar = [
        StoreRecommendation(store_id=shared_id, score=0.91),
        StoreRecommendation(store_id=similar_only_id, score=0.87),
    ]

    associated = [
        StoreRecommendation(store_id=associated_only_id, score=0.95),
        StoreRecommendation(store_id=shared_id, score=0.72),
    ]

    results = reranker.rerank_stores(
        similar_stores=similar,
        associated_stores=associated,
    )

    assert [result.store_id for result in results] == [
        shared_id,
        associated_only_id,
        similar_only_id,
    ]

    assert results[0].score == pytest.approx(0.91)


def test_rerank_products_combines_candidates_and_deduplicates_by_product_id():
    """Product candidates are fused using the same RRF strategy."""
    reranker = RRFReranker()

    shared_id = uuid4()
    similar_only_id = uuid4()

    results = reranker.rerank_products(
        similar_products=[
            ProductRecommendation(product_id=shared_id, score=0.90),
            ProductRecommendation(product_id=similar_only_id, score=0.80),
        ],
        associated_products=[
            ProductRecommendation(product_id=shared_id, score=0.70),
        ],
    )

    assert [result.product_id for result in results] == [
        shared_id,
        similar_only_id,
    ]

    assert results[0].score == pytest.approx(0.90)


def test_rerank_stores_preserves_source_order_when_candidates_have_equal_rrf_scores():
    """Equal fused scores use deterministic first-seen ordering."""
    reranker = RRFReranker()

    first_id = uuid4()
    second_id = uuid4()

    results = reranker.rerank_stores(
        similar_stores=[
            StoreRecommendation(store_id=first_id, score=0.9),
            StoreRecommendation(store_id=second_id, score=0.8),
        ],
        associated_stores=[],
    )

    assert [result.store_id for result in results] == [
        first_id,
        second_id,
    ]


def test_rerank_empty_inputs_returns_empty_results():
    """No candidates produces no recommendations."""
    reranker = RRFReranker()

    assert reranker.rerank_stores([], []) == []
    assert reranker.rerank_products([], []) == []
