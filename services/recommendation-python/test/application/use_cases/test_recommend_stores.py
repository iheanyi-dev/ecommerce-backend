"""Tests for the external Store recommendation use case."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import uuid4

from application.dto.find_similar_stores import StoreRecommendation
from application.dto.recommend_stores import RecommendStoresInput
from application.use_cases.recommend_stores import RecommendStores
from test.support.observability import StubLogger, StubMetrics


@dataclass
class StubFindSimilarStores:
    """Test double for the Store similarity candidate use case."""

    recommendations: tuple[StoreRecommendation, ...]
    calls: int = 0

    def execute(self, input):
        self.calls += 1
        self.input = input

        from application.dto.find_similar_stores import (
            FindSimilarStoresOutput,
        )

        return FindSimilarStoresOutput(
            recommendations=self.recommendations,
        )


@dataclass
class StubFindAssociatedStores:
    """Test double for the Store behavioral candidate use case."""

    recommendations: tuple[StoreRecommendation, ...]
    calls: int = 0

    def execute(self, input):
        self.calls += 1
        self.input = input

        from application.dto.find_associated_stores import (
            FindAssociatedStoresOutput,
        )

        return FindAssociatedStoresOutput(
            recommendations=self.recommendations,
        )


@dataclass
class StubReranker:
    """Test double for the Store reranking port."""

    result: list[StoreRecommendation]
    calls: int = 0

    def rerank_stores(
        self,
        similar_stores,
        associated_stores,
    ):
        self.calls += 1
        self.similar_stores = similar_stores
        self.associated_stores = associated_stores

        return self.result

    def rerank_products(
        self,
        similar_products,
        associated_products,
    ):
        raise AssertionError(
            "Product reranking should not be called by RecommendStores"
        )


def test_query_calls_only_similarity_candidate_use_case():
    store_id = uuid4()

    similar = (
        StoreRecommendation(
            store_id=store_id,
            score=0.91,
        ),
    )

    find_similar = StubFindSimilarStores(
        recommendations=similar,
    )
    find_associated = StubFindAssociatedStores(
        recommendations=(),
    )
    reranker = StubReranker(
        result=list(similar),
    )

    use_case = RecommendStores(
        find_similar_stores=find_similar,
        find_associated_stores=find_associated,
        reranker=reranker,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        RecommendStoresInput(
            query="fashion store",
        )
    )

    assert find_similar.calls == 1
    assert find_associated.calls == 0
    assert reranker.calls == 1
    assert reranker.similar_stores == list(similar)
    assert reranker.associated_stores == []
    assert output.recommendations == similar


def test_store_id_calls_only_behavioral_candidate_use_case():
    store_id = uuid4()

    associated = (
        StoreRecommendation(
            store_id=store_id,
            score=0.82,
        ),
    )

    find_similar = StubFindSimilarStores(
        recommendations=(),
    )
    find_associated = StubFindAssociatedStores(
        recommendations=associated,
    )
    reranker = StubReranker(
        result=list(associated),
    )

    use_case = RecommendStores(
        find_similar_stores=find_similar,
        find_associated_stores=find_associated,
        reranker=reranker,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        RecommendStoresInput(
            store_id=store_id,
        )
    )

    assert find_similar.calls == 0
    assert find_associated.calls == 1
    assert reranker.calls == 1
    assert reranker.similar_stores == []
    assert reranker.associated_stores == list(associated)
    assert output.recommendations == associated


def test_query_and_store_id_call_both_candidate_use_cases():
    query_store_id = uuid4()
    associated_store_id = uuid4()

    similar = (
        StoreRecommendation(
            store_id=query_store_id,
            score=0.91,
        ),
    )

    associated = (
        StoreRecommendation(
            store_id=associated_store_id,
            score=0.83,
        ),
    )

    find_similar = StubFindSimilarStores(
        recommendations=similar,
    )
    find_associated = StubFindAssociatedStores(
        recommendations=associated,
    )
    reranker = StubReranker(
        result=[associated[0], similar[0]],
    )

    use_case = RecommendStores(
        find_similar_stores=find_similar,
        find_associated_stores=find_associated,
        reranker=reranker,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        RecommendStoresInput(
            query="fashion store",
            store_id=query_store_id,
        )
    )

    assert find_similar.calls == 1
    assert find_associated.calls == 1
    assert reranker.similar_stores == list(similar)
    assert reranker.associated_stores == list(associated)
    assert output.recommendations == (associated[0], similar[0])


def test_missing_query_returns_empty_similarity_candidates():
    find_similar = StubFindSimilarStores(
        recommendations=(),
    )
    find_associated = StubFindAssociatedStores(
        recommendations=(),
    )
    reranker = StubReranker(
        result=[],
    )

    use_case = RecommendStores(
        find_similar_stores=find_similar,
        find_associated_stores=find_associated,
        reranker=reranker,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        RecommendStoresInput(
            store_id=uuid4(),
        )
    )

    assert reranker.similar_stores == []


def test_missing_store_id_returns_empty_behavioral_candidates():
    find_similar = StubFindSimilarStores(
        recommendations=(),
    )
    find_associated = StubFindAssociatedStores(
        recommendations=(),
    )
    reranker = StubReranker(
        result=[],
    )

    use_case = RecommendStores(
        find_similar_stores=find_similar,
        find_associated_stores=find_associated,
        reranker=reranker,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        RecommendStoresInput(
            query="fashion store",
        )
    )

    assert reranker.associated_stores == []


def test_no_inputs_returns_empty_candidates_to_reranker():
    find_similar = StubFindSimilarStores(
        recommendations=(),
    )
    find_associated = StubFindAssociatedStores(
        recommendations=(),
    )
    reranker = StubReranker(
        result=[],
    )

    use_case = RecommendStores(
        find_similar_stores=find_similar,
        find_associated_stores=find_associated,
        reranker=reranker,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        RecommendStoresInput()
    )

    assert find_similar.calls == 0
    assert find_associated.calls == 0
    assert reranker.calls == 1
    assert reranker.similar_stores == []
    assert reranker.associated_stores == []
    assert output.recommendations == ()
