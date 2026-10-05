"""Tests for Qdrant embedding repositories."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID, uuid4

import pytest
from qdrant_client import QdrantClient

from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)
from infrastructure.persistence.qdrant import (
    PRODUCT_COLLECTION,
    STORE_COLLECTION,
    QdrantProductEmbeddingRepository,
    QdrantStoreEmbeddingRepository,
    ensure_collection,
)


@dataclass(frozen=True, slots=True)
class RepositoryContext:
    """Test context shared by Store and Product repository tests."""

    client: QdrantClient
    repository: object
    collection_name: str
    payload_id_key: str
    entity_name: str
    id_argument: str


@pytest.fixture(
    params=[
        (
            QdrantStoreEmbeddingRepository,
            STORE_COLLECTION,
            "store_id",
            "store",
            "store_id",
        ),
        (
            QdrantProductEmbeddingRepository,
            PRODUCT_COLLECTION,
            "product_id",
            "product",
            "product_id",
        ),
    ]
)
def repository_context(request) -> RepositoryContext:
    """Create an isolated Qdrant repository for each entity type."""
    (
        repository_class,
        collection_name,
        payload_id_key,
        entity_name,
        id_argument,
    ) = request.param

    client = QdrantClient(":memory:")

    ensure_collection(
        client=client,
        collection_name=collection_name,
        vector_dimension=3,
    )

    return RepositoryContext(
        client=client,
        repository=repository_class(client),
        collection_name=collection_name,
        payload_id_key=payload_id_key,
        entity_name=entity_name,
        id_argument=id_argument,
    )


def _entity_id(entity_name: str) -> EntityID:
    """Create a domain entity identifier for a test entity."""
    return EntityID(
        value=uuid4(),
        entity_name=entity_name,
    )


def _name(entity_name: str, value: str = "Test Entity") -> EntityName:
    """Create a domain entity name for a test entity."""
    return EntityName(
        value=value,
        entity_name=entity_name,
    )


def _description(
    entity_name: str,
    value: str = "A test recommendation entity.",
) -> EntityDescription:
    """Create a domain entity description for a test entity."""
    return EntityDescription(
        value=value,
        entity_name=entity_name,
    )


def _version() -> EmbeddingModelVersion:
    """Create a test embedding model version."""
    return EmbeddingModelVersion(value="test-model-v1")


def _upsert(
    context: RepositoryContext,
    entity_id: EntityID,
    name: EntityName,
    description: EntityDescription,
    embedding: Embedding,
    embedding_model_version: EmbeddingModelVersion,
) -> None:
    """Upsert an entity using the concrete repository contract."""
    context.repository.upsert(
        **{context.id_argument: entity_id},
        name=name,
        description=description,
        embedding=embedding,
        embedding_model_version=embedding_model_version,
    )


def test_upsert_persists_vector_and_payload(
    repository_context: RepositoryContext,
) -> None:
    """An upsert stores both the vector and required entity metadata."""
    context = repository_context
    entity_id = _entity_id(context.entity_name)

    _upsert(
        context=context,
        entity_id=entity_id,
        name=_name(context.entity_name, "Coffee Store"),
        description=_description(
            context.entity_name,
            "Coffee and related products.",
        ),
        embedding=Embedding.from_values([1.0, 0.0, 0.0]),
        embedding_model_version=_version(),
    )

    points = context.client.retrieve(
        collection_name=context.collection_name,
        ids=[str(entity_id.value)],
        with_payload=True,
        with_vectors=True,
    )

    assert len(points) == 1

    point = points[0]

    assert point.id == str(entity_id.value)
    assert point.vector == [1.0, 0.0, 0.0]
    assert point.payload == {
        context.payload_id_key: str(entity_id.value),
        "name": "Coffee Store",
        "description": "Coffee and related products.",
        "embedding_model_version": "test-model-v1",
    }


def test_upsert_replaces_existing_entity(
    repository_context: RepositoryContext,
) -> None:
    """Upserting the same entity replaces its existing representation."""
    context = repository_context
    entity_id = _entity_id(context.entity_name)

    _upsert(
        context=context,
        entity_id=entity_id,
        name=_name(context.entity_name, "Original Name"),
        description=_description(
            context.entity_name,
            "Original description.",
        ),
        embedding=Embedding.from_values([1.0, 0.0, 0.0]),
        embedding_model_version=_version(),
    )

    _upsert(
        context=context,
        entity_id=entity_id,
        name=_name(context.entity_name, "Updated Name"),
        description=_description(
            context.entity_name,
            "Updated description.",
        ),
        embedding=Embedding.from_values([0.0, 1.0, 0.0]),
        embedding_model_version=EmbeddingModelVersion(
            value="test-model-v2",
        ),
    )

    points = context.client.retrieve(
        collection_name=context.collection_name,
        ids=[str(entity_id.value)],
        with_payload=True,
        with_vectors=True,
    )

    assert len(points) == 1
    assert points[0].vector == [0.0, 1.0, 0.0]
    assert points[0].payload["name"] == "Updated Name"
    assert points[0].payload["description"] == "Updated description."
    assert points[0].payload["embedding_model_version"] == "test-model-v2"


def test_delete_removes_entity_embedding(
    repository_context: RepositoryContext,
) -> None:
    """Deleting an entity removes its stored embedding."""
    context = repository_context
    entity_id = _entity_id(context.entity_name)

    _upsert(
        context=context,
        entity_id=entity_id,
        name=_name(context.entity_name),
        description=_description(context.entity_name),
        embedding=Embedding.from_values([1.0, 0.0, 0.0]),
        embedding_model_version=_version(),
    )

    context.repository.delete(entity_id)

    points = context.client.retrieve(
        collection_name=context.collection_name,
        ids=[str(entity_id.value)],
    )

    assert points == []


def test_find_similar_returns_ordered_entity_ids_and_scores(
    repository_context: RepositoryContext,
) -> None:
    """Similarity search returns Qdrant's similarity ordering."""
    context = repository_context

    closest_id = _entity_id(context.entity_name)
    second_id = _entity_id(context.entity_name)
    distant_id = _entity_id(context.entity_name)

    _upsert(
        context=context,
        entity_id=closest_id,
        name=_name(context.entity_name, "Closest"),
        description=_description(context.entity_name),
        embedding=Embedding.from_values([1.0, 0.0, 0.0]),
        embedding_model_version=_version(),
    )

    _upsert(
        context=context,
        entity_id=second_id,
        name=_name(context.entity_name, "Second"),
        description=_description(context.entity_name),
        embedding=Embedding.from_values([0.8, 0.2, 0.0]),
        embedding_model_version=_version(),
    )

    _upsert(
        context=context,
        entity_id=distant_id,
        name=_name(context.entity_name, "Distant"),
        description=_description(context.entity_name),
        embedding=Embedding.from_values([-1.0, 0.0, 0.0]),
        embedding_model_version=_version(),
    )

    results = context.repository.find_similar(
        embedding=Embedding.from_values([1.0, 0.0, 0.0]),
        limit=2,
    )

    assert len(results) == 2
    assert results[0][0] == UUID(str(closest_id.value))
    assert results[1][0] == UUID(str(second_id.value))
    assert results[0][1] > results[1][1]


def test_find_similar_rejects_non_positive_limit(
    repository_context: RepositoryContext,
) -> None:
    """Similarity searches require a positive result limit."""
    context = repository_context

    with pytest.raises(ValueError, match="limit"):
        context.repository.find_similar(
            embedding=Embedding.from_values([1.0, 0.0, 0.0]),
            limit=0,
        )


def test_ensure_collection_is_idempotent() -> None:
    """Bootstrapping an existing collection does not recreate it."""
    client = QdrantClient(":memory:")

    ensure_collection(
        client=client,
        collection_name="test_embeddings",
        vector_dimension=3,
    )

    ensure_collection(
        client=client,
        collection_name="test_embeddings",
        vector_dimension=3,
    )

    assert client.collection_exists("test_embeddings")
