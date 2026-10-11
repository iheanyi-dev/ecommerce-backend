"""Tests for ProductEmbedding."""

from uuid import uuid4

import pytest

from domain.entities import ProductEmbedding
from domain.errors import (
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
    InvalidEntityIDError,
)
from domain.value_objects import Embedding, EmbeddingModelVersion


def make_product() -> ProductEmbedding:
    return ProductEmbedding(
        id=uuid4(),
        store_id=uuid4(),
        name="Test Product",
        description="A test product.",
        embedding=Embedding.from_values([0.1, 0.2, 0.3]),
        embedding_model_version=EmbeddingModelVersion("gte-v1"),
    )


def test_product_embedding_accepts_valid_data() -> None:
    product = make_product()
    assert product.name == "Test Product"
    assert product.store_id
    assert product.embedding.dimension == 3


@pytest.mark.parametrize("field", ["id", "store_id"])
def test_product_embedding_rejects_invalid_ids(field: str) -> None:
    values = {
        "id": uuid4(),
        "store_id": uuid4(),
        "name": "Test Product",
        "description": "A test product.",
        "embedding": Embedding.from_values([0.1]),
        "embedding_model_version": EmbeddingModelVersion("gte-v1"),
    }
    values[field] = "invalid"

    with pytest.raises(InvalidEntityIDError):
        ProductEmbedding(**values)


def test_product_embedding_rejects_empty_name() -> None:
    with pytest.raises(EmptyEntityNameError):
        ProductEmbedding(
            id=uuid4(),
            store_id=uuid4(),
            name=" ",
            description="A test product.",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )


def test_product_embedding_rejects_empty_description() -> None:
    with pytest.raises(EmptyEntityDescriptionError):
        ProductEmbedding(
            id=uuid4(),
            store_id=uuid4(),
            name="Test Product",
            description=" ",
            embedding=Embedding.from_values([0.1]),
            embedding_model_version=EmbeddingModelVersion("gte-v1"),
        )
