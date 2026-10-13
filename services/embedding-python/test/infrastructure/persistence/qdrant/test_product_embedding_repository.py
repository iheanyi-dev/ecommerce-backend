"""Tests for Product embedding Qdrant persistence."""

from uuid import uuid4

import pytest
from qdrant_client import QdrantClient

from domain.entities.product_embedding import ProductEmbedding
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityID,
)
from infrastructure.persistence.qdrant.collections import (
    ensure_product_embedding_collection,
)
from infrastructure.persistence.qdrant.product_embedding_repository import (
    QdrantProductEmbeddingRepository,
)


@pytest.fixture
def repository():
    client = QdrantClient(":memory:")
    try:
        ensure_product_embedding_collection(client, 3)
        yield QdrantProductEmbeddingRepository(client)
    finally:
        client.close()


def make_product(
    store_id,
    name: str,
    vector: tuple[float, ...],
) -> ProductEmbedding:
    return ProductEmbedding(
        id=uuid4(),
        store_id=store_id,
        name=name,
        description=f"Description for {name}",
        embedding=Embedding.from_values(vector),
        embedding_model_version=EmbeddingModelVersion("test-model-v1"),
    )


def test_upsert_and_search_round_trip(repository):
    store_id = uuid4()
    product = make_product(store_id, "Product Alpha", (1.0, 0.0, 0.0))

    repository.upsert(product)
    results = repository.search(
        Embedding.from_values((1.0, 0.0, 0.0)), top_k=5, offset=0
    )

    assert len(results) == 1
    assert results[0].embedding == product
    assert results[0].score == 1.0


def test_search_can_be_scoped_to_store(repository):
    store_a = uuid4()
    store_b = uuid4()

    product_a = make_product(store_a, "Product A", (1.0, 0.0, 0.0))
    product_b = make_product(store_b, "Product B", (1.0, 0.0, 0.0))

    repository.upsert(product_a)
    repository.upsert(product_b)

    results = repository.search(
        Embedding.from_values((1.0, 0.0, 0.0)),
        top_k=5,
        offset=0,
        store_id=EntityID(store_a, "store"),
    )

    assert len(results) == 1
    assert results[0].embedding == product_a
    assert results[0].score == 1.0


def test_delete_by_store_id_only_deletes_that_stores_products(repository):
    store_a = uuid4()
    store_b = uuid4()

    product_a = make_product(store_a, "Product A", (1.0, 0.0, 0.0))
    product_b = make_product(store_b, "Product B", (0.0, 1.0, 0.0))

    repository.upsert(product_a)
    repository.upsert(product_b)

    repository.delete_by_store_id(EntityID(store_a, "store"))

    results = repository.search(
        Embedding.from_values((1.0, 0.0, 0.0)), top_k=5, offset=0
    )

    assert len(results) == 1
    assert results[0].embedding == product_b
    assert results[0].score == 0.0


def test_delete_is_idempotent(repository):
    store_id = uuid4()
    product = make_product(store_id, "Product Alpha", (1.0, 0.0, 0.0))
    product_id = EntityID(product.id, "product")

    repository.upsert(product)
    repository.delete(product_id)
    repository.delete(product_id)

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
