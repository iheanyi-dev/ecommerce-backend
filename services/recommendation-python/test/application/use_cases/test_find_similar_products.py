"""Tests for the FindSimilarProducts use case."""

from __future__ import annotations

from uuid import UUID, uuid4

import pytest

from application.dto import (
    FindSimilarProductsInput,
)
from application.use_cases import FindSimilarProducts
from test.support.observability import StubLogger, StubMetrics
from domain.value_objects import Embedding


class StubEmbeddingGenerator:
    """Fake embedding generator for application-level tests."""

    def __init__(self) -> None:
        self.queries: list[str] = []

    def generate_product_query_embedding(
        self,
        query: str,
    ) -> Embedding:
        self.queries.append(query)

        return Embedding.from_values(
            (0.1, 0.2, 0.3),
        )


class StubProductEmbeddingRepository:
    """Fake Product embedding repository for application-level tests."""

    def __init__(
        self,
        recommendations: list[tuple[UUID, float]],
    ) -> None:
        self.recommendations = recommendations
        self.calls: list[tuple[Embedding, int]] = []

    def find_similar(
        self,
        embedding: Embedding,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        self.calls.append((embedding, limit))

        return self.recommendations[:limit]


def make_recommendations(
    count: int,
) -> list[tuple[UUID, float]]:
    """Create deterministic recommendation candidates for tests."""

    return [
        (uuid4(), 1.0 - (index * 0.01))
        for index in range(count)
    ]


def test_generates_product_query_embedding() -> None:
    """The use case generates an embedding from the supplied query."""

    embedding_generator = StubEmbeddingGenerator()
    repository = StubProductEmbeddingRepository(
        make_recommendations(10),
    )

    use_case = FindSimilarProducts(
        embedding_generator=embedding_generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    use_case.execute(
        FindSimilarProductsInput(
            query="wireless headphones",
            page=1,
        )
    )

    assert embedding_generator.queries == [
        "wireless headphones",
    ]


def test_page_two_requests_twenty_candidates_and_returns_second_page() -> None:
    """Page two retrieves twenty candidates and returns candidates 10-19."""

    embedding_generator = StubEmbeddingGenerator()
    recommendations = make_recommendations(20)
    repository = StubProductEmbeddingRepository(recommendations)

    use_case = FindSimilarProducts(
        embedding_generator=embedding_generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindSimilarProductsInput(
            query="wireless headphones",
            page=2,
        )
    )

    assert repository.calls[0][1] == 20

    assert len(output.recommendations) == 10

    assert [
        recommendation.product_id
        for recommendation in output.recommendations
    ] == [
        product_id
        for product_id, _ in recommendations[10:20]
    ]


def test_last_page_returns_remaining_candidates() -> None:
    """A final partial page returns all candidates remaining after its offset."""

    embedding_generator = StubEmbeddingGenerator()
    recommendations = make_recommendations(13)
    repository = StubProductEmbeddingRepository(recommendations)

    use_case = FindSimilarProducts(
        embedding_generator=embedding_generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindSimilarProductsInput(
            query="wireless headphones",
            page=2,
        )
    )

    assert [
        recommendation.product_id
        for recommendation in output.recommendations
    ] == [
        product_id
        for product_id, _ in recommendations[10:]
    ]


def test_page_zero_is_rejected() -> None:
    """Recommendation pages are one-based."""

    embedding_generator = StubEmbeddingGenerator()
    repository = StubProductEmbeddingRepository([])

    use_case = FindSimilarProducts(
        embedding_generator=embedding_generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    with pytest.raises(
        ValueError,
        match="page must be greater than or equal to 1",
    ):
        use_case.execute(
            FindSimilarProductsInput(
                query="wireless headphones",
                page=0,
            )
        )


def test_page_one_returns_no_more_than_page_size() -> None:
    """The first recommendation page contains at most ten results."""

    embedding_generator = StubEmbeddingGenerator()
    recommendations = make_recommendations(20)
    repository = StubProductEmbeddingRepository(recommendations)

    use_case = FindSimilarProducts(
        embedding_generator=embedding_generator,
        product_embedding_repository=repository,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    output = use_case.execute(
        FindSimilarProductsInput(
            query="wireless headphones",
            page=1,
        )
    )

    assert len(output.recommendations) == 10
    assert repository.calls[0][1] == 10
