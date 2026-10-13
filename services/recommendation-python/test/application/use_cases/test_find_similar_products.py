"""Tests for the FindSimilarProducts use case."""

from uuid import uuid4

import pytest

from application.dto import FindSimilarProductsInput, ProductRecommendation
from application.use_cases import FindSimilarProducts
from test.support.observability import StubLogger, StubMetrics


class StubSimilarityRetriever:
    """Fake Embedding retrieval port."""

    def __init__(self, products=None):
        self.products = products or []
        self.product_calls = []
        self.store_calls = []

    def find_similar_products(self, query, top_k, offset):
        self.product_calls.append((query, top_k, offset))
        return self.products[offset:offset + top_k]

    def find_similar_stores(self, query, top_k, offset):
        self.store_calls.append((query, top_k, offset))
        return []


def test_delegates_query_and_page_window_to_embedding():
    candidates = [(uuid4(), 0.9), (uuid4(), 0.8)]
    retriever = StubSimilarityRetriever(candidates)
    use_case = FindSimilarProducts(
        similarity_retriever=retriever,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindSimilarProductsInput(query="wireless headphones", page=1)
    )

    assert retriever.product_calls == [
        ("wireless headphones", 10, 0)
    ]
    assert output.recommendations == tuple(
        ProductRecommendation(product_id=product_id, score=score)
        for product_id, score in candidates
    )


def test_page_two_uses_remote_offset_and_page_size():
    candidates = [(uuid4(), 0.7), (uuid4(), 0.6)]
    retriever = StubSimilarityRetriever(candidates)
    use_case = FindSimilarProducts(
        similarity_retriever=retriever,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        FindSimilarProductsInput(query="headphones", page=2)
    )

    assert retriever.product_calls == [("headphones", 10, 10)]


def test_rejects_invalid_page():
    use_case = FindSimilarProducts(
        similarity_retriever=StubSimilarityRetriever(),
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(
        ValueError,
        match="page must be greater than or equal to 1",
    ):
        use_case.execute(FindSimilarProductsInput(query="headphones", page=0))
