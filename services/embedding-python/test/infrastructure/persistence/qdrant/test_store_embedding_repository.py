"""Tests for Store embedding Qdrant persistence."""

from uuid import uuid4

import pytest
from qdrant_client import QdrantClient

from domain.entities.store_embedding import StoreEmbedding
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityID,
)
from infrastructure.persistence.qdrant.collections import (
    ensure_store_embedding_collection,
)
from infrastructure.persistence.qdrant.store_embedding_repository import (
    QdrantStoreEmbeddingRepository,
)


@pytest.fixture
def repository():
    client = QdrantClient(":memory:")
    try:
        ensure_store_embedding_collection(client, 3)
        yield QdrantStoreEmbeddingRepository(client)
    finally:
        client.close()


def make_store(name: str, vector: tuple[float, ...]) -> StoreEmbedding:
    return StoreEmbedding(
        id=uuid4(),
        name=name,
        description=f"Description for {name}",
        embedding=Embedding.from_values(vector),
        embedding_model_version=EmbeddingModelVersion("test-model-v1"),
    )


def test_upsert_and_search_round_trip(repository):
    store = make_store("Store Alpha", (1.0, 0.0, 0.0))

    repository.upsert(store)
    results = repository.search(
        Embedding.from_values((1.0, 0.0, 0.0)), top_k=5, offset=0
    )

    assert len(results) == 1
    assert results[0] == store


def test_upsert_replaces_existing_store(repository):
    store = make_store("Store Alpha", (1.0, 0.0, 0.0))
    repository.upsert(store)

    updated = StoreEmbedding(
        id=store.id,
        name="Store Updated",
        description="Updated description",
        embedding=Embedding.from_values((0.0, 1.0, 0.0)),
        embedding_model_version=EmbeddingModelVersion("test-model-v2"),
    )
    repository.upsert(updated)

    results = repository.search(
        Embedding.from_values((0.0, 1.0, 0.0)), top_k=5, offset=0
    )

    assert results == [updated]


def test_delete_is_idempotent(repository):
    store = make_store("Store Alpha", (1.0, 0.0, 0.0))
    store_id = EntityID(store.id, "store")

    repository.upsert(store)
    repository.delete(store_id)
    repository.delete(store_id)

    assert repository.search(
        Embedding.from_values((1.0, 0.0, 0.0)), top_k=5, offset=0
    ) == []


@pytest.mark.parametrize(
    ("top_k", "offset"),
    [(0, 0), (-1, 0), (True, 0), (1, -1), (1, True)],
)
def test_search_rejects_invalid_pagination(repository, top_k, offset):
    with pytest.raises(ValueError):
        repository.search(
            Embedding.from_values((1.0, 0.0, 0.0)),
            top_k=top_k,
            offset=offset,
        )
