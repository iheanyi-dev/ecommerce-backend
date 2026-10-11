"""Tests for KnowledgeChunk."""

from uuid import uuid4

import pytest

from domain.entities import KnowledgeChunk
from domain.errors import (
    EmptyKnowledgeChunkContentError,
    EmptyStorageReferenceError,
    InvalidChunkIndexError,
    InvalidEntityIDError,
)
from domain.value_objects import Embedding, EmbeddingModelVersion


def make_chunk() -> KnowledgeChunk:
    return KnowledgeChunk(
        id=uuid4(),
        knowledge_id=uuid4(),
        store_id=uuid4(),
        storage_reference="knowledge/store-id/guide.pdf",
        content="A useful knowledge chunk.",
        chunk_index=0,
        embedding=Embedding.from_values([0.1, 0.2, 0.3]),
        embedding_model_version=EmbeddingModelVersion("gte-v1"),
    )


def test_knowledge_chunk_accepts_valid_data() -> None:
    chunk = make_chunk()
    assert chunk.content == "A useful knowledge chunk."
    assert chunk.chunk_index == 0
    assert chunk.embedding.dimension == 3


@pytest.mark.parametrize("field", ["id", "knowledge_id", "store_id"])
def test_knowledge_chunk_rejects_invalid_ids(field: str) -> None:
    values = {
        "id": uuid4(),
        "knowledge_id": uuid4(),
        "store_id": uuid4(),
        "storage_reference": "knowledge/store-id/guide.pdf",
        "content": "Useful content.",
        "chunk_index": 0,
        "embedding": Embedding.from_values([0.1]),
        "embedding_model_version": EmbeddingModelVersion("gte-v1"),
    }
    values[field] = "invalid"

    with pytest.raises(InvalidEntityIDError):
        KnowledgeChunk(**values)


def test_knowledge_chunk_rejects_empty_content() -> None:
    with pytest.raises(EmptyKnowledgeChunkContentError):
        KnowledgeChunk(
            id=uuid4(),
            knowledge_id=uuid4(),
            store_id=uuid4(),
            storage_reference="knowledge/store-id/guide.pdf",
            content=" ",
            chunk_index=0,
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )


def test_knowledge_chunk_rejects_empty_storage_reference() -> None:
    with pytest.raises(EmptyStorageReferenceError):
        KnowledgeChunk(
            id=uuid4(),
            knowledge_id=uuid4(),
            store_id=uuid4(),
            storage_reference=" ",
            content="Useful content.",
            chunk_index=0,
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )


@pytest.mark.parametrize("index", [-1, 1.5, True])
def test_knowledge_chunk_rejects_invalid_chunk_index(index: int) -> None:
    with pytest.raises(InvalidChunkIndexError):
        KnowledgeChunk(
            id=uuid4(),
            knowledge_id=uuid4(),
            store_id=uuid4(),
            storage_reference="knowledge/store-id/guide.pdf",
            content="Useful content.",
            chunk_index=index,
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )
