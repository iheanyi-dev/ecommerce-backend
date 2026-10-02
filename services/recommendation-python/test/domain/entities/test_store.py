"""Tests for the Store domain entity."""

from uuid import uuid4

import pytest

from domain.entities.store import Store
from domain.errors import (
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
    InvalidEntityIDError,
)
from domain.value_objects.embedding import Embedding
from domain.value_objects.embedding_model_version import EmbeddingModelVersion


def make_store() -> Store:
    """Create a valid Store for domain tests."""
    return Store(
        id=uuid4(),
        name="Test Store",
        description="A test store.",
        embedding=Embedding.from_values([0.1, 0.2, 0.3]),
        embedding_model_version=EmbeddingModelVersion("store-v1"),
    )


def test_store_accepts_valid_data() -> None:
    """A Store must accept valid recommendation representation data."""
    store = make_store()

    assert store.name == "Test Store"
    assert store.description == "A test store."
    assert store.embedding.dimension == 3


def test_store_rejects_invalid_id() -> None:
    """A Store must use UUID identity."""
    with pytest.raises(InvalidEntityIDError):
        Store(
            id="not-a-uuid",
            name="Test Store",
            description="A test store.",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("store-v1"),
        )


def test_store_rejects_empty_name() -> None:
    """A Store name cannot be empty."""
    with pytest.raises(EmptyEntityNameError):
        Store(
            id=uuid4(),
            name="   ",
            description="A test store.",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("store-v1"),
        )


def test_store_rejects_empty_description() -> None:
    """A Store description cannot be empty."""
    with pytest.raises(EmptyEntityDescriptionError):
        Store(
            id=uuid4(),
            name="Test Store",
            description="   ",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("store-v1"),
        )
