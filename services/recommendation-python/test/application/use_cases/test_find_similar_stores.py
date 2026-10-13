"""Tests for the FindSimilarStores use case."""

from uuid import uuid4

import pytest

from application.dto import FindSimilarStoresInput, StoreRecommendation
from application.use_cases import FindSimilarStores
from test.support.observability import StubLogger, StubMetrics


class StubSimilarityRetriever:
    """Fake Embedding retrieval port."""

    def __init__(self, stores=None):
        self.stores = stores or []
        self.store_calls = []

    def find_similar_products(self, query, top_k, offset):
        return []

    def find_similar_stores(self, query, top_k, offset):
        self.store_calls.append((query, top_k, offset))
        return self.stores[offset:offset + top_k]


def test_delegates_query_and_page_window_to_embedding():
    candidates = [(uuid4(), 0.95), (uuid4(), 0.85)]
    retriever = StubSimilarityRetriever(candidates)
    use_case = FindSimilarStores(
        similarity_retriever=retriever,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindSimilarStoresInput(query="fashion stores", page=1)
    )

    assert retriever.store_calls == [("fashion stores", 10, 0)]
    assert output.recommendations == tuple(
        StoreRecommendation(store_id=store_id, score=score)
        for store_id, score in candidates
    )


def test_page_two_uses_remote_offset_and_page_size():
    retriever = StubSimilarityRetriever()
    use_case = FindSimilarStores(
        similarity_retriever=retriever,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(FindSimilarStoresInput(query="fashion", page=2))

    assert retriever.store_calls == [("fashion", 10, 10)]


def test_rejects_invalid_page():
    use_case = FindSimilarStores(
        similarity_retriever=StubSimilarityRetriever(),
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(ValueError):
        use_case.execute(FindSimilarStoresInput(query="fashion", page=0))
