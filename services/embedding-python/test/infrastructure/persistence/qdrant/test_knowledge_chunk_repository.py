"""Unit tests use mocks only and never write to a Qdrant collection."""

from types import SimpleNamespace
from unittest.mock import MagicMock
from uuid import uuid4

import pytest
from qdrant_client import models

from domain.entities.knowledge_chunk import KnowledgeChunk
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityID,
)
from infrastructure.persistence.qdrant import (
    KNOWLEDGE_CHUNK_COLLECTION,
    QdrantKnowledgeChunkRepository,
    ensure_knowledge_chunk_collection,
)


def make_chunk(
    *,
    knowledge_id=None,
    store_id=None,
    chunk_id=None,
    content="A sample knowledge chunk",
    chunk_index=0,
):
    return KnowledgeChunk(
        id=chunk_id or uuid4(),
        knowledge_id=knowledge_id or uuid4(),
        store_id=store_id or uuid4(),
        storage_reference="knowledge/example/document.txt",
        content=content,
        chunk_index=chunk_index,
        embedding=Embedding.from_values([0.6, 0.8]),
        embedding_model_version=EmbeddingModelVersion("test-model@test-revision"),
    )


def make_repository():
    # A mock client prevents unit tests from connecting to real Qdrant.
    client = MagicMock()
    repository = QdrantKnowledgeChunkRepository(client)
    return repository, client


def test_uses_dedicated_knowledge_collection_by_default():
    repository, _ = make_repository()

    assert repository._collection_name == KNOWLEDGE_CHUNK_COLLECTION
    assert KNOWLEDGE_CHUNK_COLLECTION == "knowledge_chunks"


def test_replace_upserts_points_with_required_payload_and_waits():
    repository, client = make_repository()
    knowledge_id = uuid4()
    store_id = uuid4()
    chunk = make_chunk(knowledge_id=knowledge_id, store_id=store_id)
    client.scroll.return_value = ([], None)

    repository.replace_for_knowledge(
        EntityID(knowledge_id, "knowledge"),
        [chunk],
    )

    client.upsert.assert_called_once()
    call = client.upsert.call_args.kwargs
    assert call["collection_name"] == KNOWLEDGE_CHUNK_COLLECTION
    assert call["wait"] is True

    point = call["points"][0]
    assert point.id == str(chunk.id)
    assert point.vector == [0.6, 0.8]
    assert point.payload == {
        "knowledge_id": str(knowledge_id),
        "store_id": str(store_id),
        "storage_reference": chunk.storage_reference,
        "content": chunk.content,
        "chunk_index": 0,
        "embedding_model_version": "test-model@test-revision",
    }
    client.delete.assert_not_called()


def test_replace_deletes_stale_ids_only_after_upsert():
    repository, client = make_repository()
    knowledge_id = uuid4()
    old_id = uuid4()
    chunk = make_chunk(knowledge_id=knowledge_id)
    client.scroll.return_value = (
        [SimpleNamespace(id=old_id)],
        None,
    )
    call_order = []
    client.upsert.side_effect = lambda **kwargs: call_order.append("upsert")
    client.delete.side_effect = lambda **kwargs: call_order.append("delete")

    repository.replace_for_knowledge(
        EntityID(knowledge_id, "knowledge"),
        [chunk],
    )

    assert call_order == ["upsert", "delete"]
    selector = client.delete.call_args.kwargs["points_selector"]
    assert set(selector.points) == {str(old_id)}


def test_failed_upsert_does_not_delete_existing_points():
    repository, client = make_repository()
    knowledge_id = uuid4()
    client.scroll.return_value = ([SimpleNamespace(id=uuid4())], None)
    client.upsert.side_effect = RuntimeError("write failed")
    chunk = make_chunk(knowledge_id=knowledge_id)

    with pytest.raises(RuntimeError, match="write failed"):
        repository.replace_for_knowledge(
            EntityID(knowledge_id, "knowledge"),
            [chunk],
        )

    client.delete.assert_not_called()


def test_empty_replacement_deletes_document_points_by_filter():
    repository, client = make_repository()
    knowledge_id = uuid4()

    repository.replace_for_knowledge(
        EntityID(knowledge_id, "knowledge"),
        [],
    )

    client.upsert.assert_not_called()
    selector = client.delete.call_args.kwargs["points_selector"]
    assert isinstance(selector, models.FilterSelector)
    condition = selector.filter.must[0]
    assert condition.key == "knowledge_id"
    assert condition.match.value == str(knowledge_id)


def test_delete_by_knowledge_id_is_idempotent_filter_delete():
    repository, client = make_repository()
    knowledge_id = uuid4()

    repository.delete_by_knowledge_id(
        EntityID(knowledge_id, "knowledge")
    )
    repository.delete_by_knowledge_id(
        EntityID(knowledge_id, "knowledge")
    )

    assert client.delete.call_count == 2
    for call in client.delete.call_args_list:
        selector = call.kwargs["points_selector"]
        assert isinstance(selector, models.FilterSelector)
        assert selector.filter.must[0].key == "knowledge_id"


def test_search_applies_store_filter_and_reconstructs_chunk():
    repository, client = make_repository()
    chunk = make_chunk()
    client.query_points.return_value = SimpleNamespace(
        points=[
            SimpleNamespace(
                id=str(chunk.id),
                payload={
                    "knowledge_id": str(chunk.knowledge_id),
                    "store_id": str(chunk.store_id),
                    "storage_reference": chunk.storage_reference,
                    "content": chunk.content,
                    "chunk_index": chunk.chunk_index,
                    "embedding_model_version": str(
                        chunk.embedding_model_version
                    ),
                },
                vector=list(chunk.embedding.vector),
                    score=0.875,
            )
        ]
    )

    results = repository.search(
        query_embedding=Embedding.from_values([1.0, 0.0]),
        top_k=5,
        offset=2,
        store_id=EntityID(chunk.store_id, "store"),
    )

    call = client.query_points.call_args.kwargs
    assert call["collection_name"] == KNOWLEDGE_CHUNK_COLLECTION
    assert call["limit"] == 5
    assert call["offset"] == 2
    assert call["with_payload"] is True
    assert call["with_vectors"] is True
    store_condition = call["query_filter"].must[0]
    assert store_condition.key == "store_id"
    assert store_condition.match.value == str(chunk.store_id)

    assert len(results) == 1
    restored = results[0].chunk
    assert results[0].score == 0.875
    assert restored.id == chunk.id
    assert restored.knowledge_id == chunk.knowledge_id
    assert restored.store_id == chunk.store_id
    assert restored.content == chunk.content
    assert restored.embedding.vector == chunk.embedding.vector
    assert restored.embedding_model_version == chunk.embedding_model_version


@pytest.mark.parametrize(
    ("top_k", "offset"),
    [(0, 0), (-1, 0), (True, 0), (1, -1), (1, True)],
)
def test_search_rejects_invalid_pagination(top_k, offset):
    repository, client = make_repository()

    with pytest.raises(ValueError):
        repository.search(
            query_embedding=Embedding.from_values([1.0, 0.0]),
            top_k=top_k,
            offset=offset,
            store_id=EntityID(uuid4(), "store"),
        )

    client.query_points.assert_not_called()


def test_replace_rejects_chunks_from_different_stores_before_writing():
    repository, client = make_repository()
    knowledge_id = uuid4()
    chunks = [
        make_chunk(knowledge_id=knowledge_id, store_id=uuid4()),
        make_chunk(knowledge_id=knowledge_id, store_id=uuid4()),
    ]

    with pytest.raises(ValueError, match="one store"):
        repository.replace_for_knowledge(
            EntityID(knowledge_id, "knowledge"),
            chunks,
        )

    client.upsert.assert_not_called()
    client.delete.assert_not_called()


def test_replace_rejects_chunks_for_another_document_before_writing():
    repository, client = make_repository()

    with pytest.raises(ValueError, match="knowledge ID"):
        repository.replace_for_knowledge(
            EntityID(uuid4(), "knowledge"),
            [make_chunk()],
        )

    client.upsert.assert_not_called()
    client.delete.assert_not_called()


def test_collection_bootstrap_creates_only_dedicated_collection():
    client = MagicMock()
    client.collection_exists.return_value = False

    ensure_knowledge_chunk_collection(client, vector_dimension=768)

    client.create_collection.assert_called_once()
    call = client.create_collection.call_args.kwargs
    assert call["collection_name"] == KNOWLEDGE_CHUNK_COLLECTION
    assert call["vectors_config"].size == 768
    assert call["vectors_config"].distance == models.Distance.COSINE


def test_collection_bootstrap_does_not_recreate_existing_collection():
    client = MagicMock()
    client.collection_exists.return_value = True

    ensure_knowledge_chunk_collection(client, vector_dimension=768)

    client.create_collection.assert_not_called()


@pytest.mark.parametrize("dimension", [0, -1, True, 1.5])
def test_collection_bootstrap_rejects_invalid_dimension(dimension):
    client = MagicMock()

    with pytest.raises(ValueError):
        ensure_knowledge_chunk_collection(client, dimension)

    client.collection_exists.assert_not_called()
    client.create_collection.assert_not_called()
