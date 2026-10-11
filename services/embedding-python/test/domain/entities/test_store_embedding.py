"""Tests for StoreEmbedding."""

from uuid import uuid4

import pytest

from domain.entities import StoreEmbedding
from domain.errors import (
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
    InvalidEntityIDError,
)
from domain.value_objects import Embedding, EmbeddingModelVersion


def make_store() -> StoreEmbedding:
    return StoreEmbedding(
        id=uuid4(),
        name="Test Store",
        description="A test store.",
        embedding=Embedding.from_values([0.1, 0.2, 0.3]),
        embedding_model_version=EmbeddingModelVersion("gte-v1"),
    )


def test_store_embedding_accepts_valid_data() -> None:
    store = make_store()
    assert store.name == "Test Store"
    assert store.description == "A test store."
    assert store.embedding.dimension == 3


def test_store_embedding_rejects_invalid_id() -> None:
    with pytest.raises(InvalidEntityIDError):
        StoreEmbedding(
            id="invalid",
            name="Test Store",
            description="A test store.",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )


def test_store_embedding_rejects_empty_name() -> None:
    with pytest.raises(EmptyEntityNameError):
        StoreEmbedding(
            id=uuid4(),
            name=" ",
            description="A test store.",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )


def test_store_embedding_rejects_empty_description() -> None:
    with pytest.raises(EmptyEntityDescriptionError):
        StoreEmbedding(
            id=uuid4(),
            name="Test Store",
            description=" ",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )
