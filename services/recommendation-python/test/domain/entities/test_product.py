"""Tests for the Product domain entity."""

from uuid import uuid4

import pytest

from domain.entities.product import Product
from domain.errors import (
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
    InvalidEntityIDError,
)
from domain.value_objects.embedding import Embedding
from domain.value_objects.embedding_model_version import EmbeddingModelVersion


def make_product() -> Product:
    """Create a valid Product for domain tests."""
    return Product(
        id=uuid4(),
        name="Test Product",
        description="A test product.",
        embedding=Embedding.from_values([0.1, 0.2, 0.3]),
        embedding_model_version=EmbeddingModelVersion("product-v1"),
    )


def test_product_accepts_valid_data() -> None:
    """A Product must accept valid recommendation representation data."""
    product = make_product()

    assert product.name == "Test Product"
    assert product.description == "A test product."
    assert product.embedding.dimension == 3


def test_product_rejects_invalid_id() -> None:
    """A Product must use UUID identity."""
    with pytest.raises(InvalidEntityIDError):
        Product(
            id="not-a-uuid",
            name="Test Product",
            description="A test product.",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("product-v1"),
        )


def test_product_rejects_empty_name() -> None:
    """A Product name cannot be empty."""
    with pytest.raises(EmptyEntityNameError):
        Product(
            id=uuid4(),
            name="   ",
            description="A test product.",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("product-v1"),
        )


def test_product_rejects_empty_description() -> None:
    """A Product description cannot be empty."""
    with pytest.raises(EmptyEntityDescriptionError):
        Product(
            id=uuid4(),
            name="Test Product",
            description="   ",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("product-v1"),
        )
