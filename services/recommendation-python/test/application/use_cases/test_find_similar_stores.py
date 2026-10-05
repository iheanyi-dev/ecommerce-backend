"""Tests for the FindSimilarStores use case."""

from uuid import UUID, uuid4

import pytest

from application.dto import (
    FindSimilarStoresInput,
    StoreRecommendation,
)
from application.use_cases import FindSimilarStores
from domain.value_objects import Embedding


class StubEmbeddingGenerator:
    """Test double for the embedding generator."""

    def __init__(self) -> None:
        self.queries: list[str] = []

    def generate_store_query_embedding(
        self,
        query: str,
    ) -> Embedding:
        self.queries.append(query)

        return Embedding.from_values((0.1, 0.2, 0.3))


class StubStoreEmbeddingRepository:
    """Test double for Store similarity search."""

    def __init__(
        self,
        recommendations: list[tuple[UUID, float]],
    ) -> None:
        self.recommendations = recommendations
        self.calls: list[dict[str, object]] = []

    def find_similar(
        self,
        embedding: Embedding,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        self.calls.append(
            {
                "embedding": embedding,
                "limit": limit,
            }
        )

        return self.recommendations[:limit]


def test_find_similar_stores_generates_query_embedding() -> None:
    generator = StubEmbeddingGenerator()
    repository = StubStoreEmbeddingRepository([])

    use_case = FindSimilarStores(
        embedding_generator=generator,
        store_embedding_repository=repository,
    )

    use_case.execute(
        FindSimilarStoresInput(
            query="fashion stores",
            page=1,
        )
    )

    assert generator.queries == ["fashion stores"]


def test_find_similar_stores_uses_page_as_result_window() -> None:
    recommendations = [
        (uuid4(), 0.99),
        (uuid4(), 0.95),
        (uuid4(), 0.91),
        (uuid4(), 0.87),
        (uuid4(), 0.83),
        (uuid4(), 0.79),
        (uuid4(), 0.75),
        (uuid4(), 0.71),
        (uuid4(), 0.67),
        (uuid4(), 0.63),
        (uuid4(), 0.59),
        (uuid4(), 0.55),
        (uuid4(), 0.51),
        (uuid4(), 0.47),
        (uuid4(), 0.43),
        (uuid4(), 0.39),
        (uuid4(), 0.35),
        (uuid4(), 0.31),
        (uuid4(), 0.27),
        (uuid4(), 0.23),
    ]

    generator = StubEmbeddingGenerator()
    repository = StubStoreEmbeddingRepository(recommendations)

    use_case = FindSimilarStores(
        embedding_generator=generator,
        store_embedding_repository=repository,
    )

    output = use_case.execute(
        FindSimilarStoresInput(
            query="fashion stores",
            page=2,
        )
    )

    assert repository.calls[0]["limit"] == 20

    assert output.recommendations == tuple(
        StoreRecommendation(store_id=store_id, score=score)
        for store_id, score in recommendations[10:20]
    )


def test_find_similar_stores_returns_remaining_results_on_last_page() -> None:
    recommendations = [
        (uuid4(), 0.99),
        (uuid4(), 0.95),
        (uuid4(), 0.91),
        (uuid4(), 0.87),
        (uuid4(), 0.83),
        (uuid4(), 0.79),
        (uuid4(), 0.75),
        (uuid4(), 0.71),
        (uuid4(), 0.67),
        (uuid4(), 0.63),
        (uuid4(), 0.59),
        (uuid4(), 0.55),
    ]

    generator = StubEmbeddingGenerator()
    repository = StubStoreEmbeddingRepository(recommendations)

    use_case = FindSimilarStores(
        embedding_generator=generator,
        store_embedding_repository=repository,
    )

    output = use_case.execute(
        FindSimilarStoresInput(
            query="fashion stores",
            page=2,
        )
    )

    assert output.recommendations == tuple(
        StoreRecommendation(store_id=store_id, score=score)
        for store_id, score in recommendations[10:]
    )


def test_find_similar_stores_rejects_invalid_page() -> None:
    use_case = FindSimilarStores(
        embedding_generator=StubEmbeddingGenerator(),
        store_embedding_repository=StubStoreEmbeddingRepository([]),
    )

    with pytest.raises(ValueError):
        use_case.execute(
            FindSimilarStoresInput(
                query="fashion stores",
                page=0,
            )
        )


def test_find_similar_stores_does_not_return_more_than_page_size() -> None:
    recommendations = [
        (uuid4(), 1.0 - (index * 0.01))
        for index in range(15)
    ]

    generator = StubEmbeddingGenerator()
    repository = StubStoreEmbeddingRepository(recommendations)

    use_case = FindSimilarStores(
        embedding_generator=generator,
        store_embedding_repository=repository,
    )

    output = use_case.execute(
        FindSimilarStoresInput(
            query="fashion stores",
            page=1,
        )
    )

    assert len(output.recommendations) == 10
