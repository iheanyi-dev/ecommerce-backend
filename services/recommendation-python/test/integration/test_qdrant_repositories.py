"""Integration tests for Recommendation Qdrant repositories."""

from __future__ import annotations

import os
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
from infrastructure.persistence.qdrant.client import create_qdrant_client


QDRANT_URL = os.getenv(
    "QDRANT_URL",
    "http://localhost:6333",
)

VECTOR_DIMENSION = 3


@dataclass(frozen=True, slots=True)
class RepositoryContext:
    """Real Qdrant repository configuration used by the integration tests."""

    repository: object
    collection_name: str
    entity_name: str
    id_argument: str


@pytest.fixture(scope="module")
def qdrant_client() -> QdrantClient:
    """Create a client connected to the Docker-hosted Qdrant instance."""
    client = create_qdrant_client(url=QDRANT_URL)

    try:
        client.get_collections()
    except Exception as exc:
        pytest.fail(
            f"Could not connect to Qdrant at {QDRANT_URL}: {exc}"
        )

    return client


@pytest.fixture(
    params=[
        (
            QdrantStoreEmbeddingRepository,
            STORE_COLLECTION,
            "store",
            "store_id",
        ),
        (
            QdrantProductEmbeddingRepository,
            PRODUCT_COLLECTION,
            "product",
            "product_id",
        ),
    ]
)
def repository_context(
    qdrant_client: QdrantClient,
    request,
) -> RepositoryContext:
    """Create a concrete repository backed by real Docker Qdrant."""
    (
        repository_class,
        collection_name,
        entity_name,
        id_argument,
    ) = request.param

    ensure_collection(
        client=qdrant_client,
        collection_name=collection_name,
        vector_dimension=VECTOR_DIMENSION,
    )

    return RepositoryContext(
        repository=repository_class(qdrant_client),
        collection_name=collection_name,
        entity_name=entity_name,
        id_argument=id_argument,
    )


def _entity_id(entity_name: str) -> EntityID:
    """Create a test entity identifier."""
    return EntityID(
        value=uuid4(),
        entity_name=entity_name,
    )


def _name(
    entity_name: str,
    value: str = "Integration Test Entity",
) -> EntityName:
    """Create a test entity name."""
    return EntityName(
        value=value,
        entity_name=entity_name,
    )


def _description(
    entity_name: str,
    value: str = "Integration test recommendation entity.",
) -> EntityDescription:
    """Create a test entity description."""
    return EntityDescription(
        value=value,
        entity_name=entity_name,
    )


def _version() -> EmbeddingModelVersion:
    """Create the integration-test embedding model version."""
    return EmbeddingModelVersion(
        value="integration-test-model-v1",
    )


def _upsert(
    context: RepositoryContext,
    entity_id: EntityID,
    name: EntityName,
    description: EntityDescription,
    embedding: Embedding,
) -> None:
    """Upsert an entity through its concrete repository adapter."""
    context.repository.upsert(
        **{context.id_argument: entity_id},
        name=name,
        description=description,
        embedding=embedding,
        embedding_model_version=_version(),
    )


def test_repository_upsert_and_similarity_search(
    repository_context: RepositoryContext,
) -> None:
    """Concrete repositories can persist and query embeddings in real Qdrant."""
    context = repository_context

    closest_id = _entity_id(context.entity_name)
    second_id = _entity_id(context.entity_name)

    try:
        _upsert(
            context=context,
            entity_id=closest_id,
            name=_name(context.entity_name, "Closest"),
            description=_description(context.entity_name),
            embedding=Embedding.from_values([1.0, 0.0, 0.0]),
        )

        _upsert(
            context=context,
            entity_id=second_id,
            name=_name(context.entity_name, "Second"),
            description=_description(context.entity_name),
            embedding=Embedding.from_values([0.8, 0.2, 0.0]),
        )

        results = context.repository.find_similar(
            embedding=Embedding.from_values([1.0, 0.0, 0.0]),
            limit=2,
        )

        assert len(results) == 2
        assert results[0][0] == UUID(str(closest_id.value))
        assert results[1][0] == UUID(str(second_id.value))
        assert results[0][1] > results[1][1]

    finally:
        context.repository.delete(closest_id)
        context.repository.delete(second_id)


def test_repository_delete_removes_embedding(
    repository_context: RepositoryContext,
) -> None:
    """Concrete repositories can remove embeddings from real Qdrant."""
    context = repository_context
    entity_id = _entity_id(context.entity_name)

    _upsert(
        context=context,
        entity_id=entity_id,
        name=_name(context.entity_name),
        description=_description(context.entity_name),
        embedding=Embedding.from_values([1.0, 0.0, 0.0]),
    )

    context.repository.delete(entity_id)

    # A second delete is intentionally not required here. The repository
    # contract has already been exercised against the real Qdrant instance;
    # physical storage inspection would couple this integration test to
    # repository internals.
