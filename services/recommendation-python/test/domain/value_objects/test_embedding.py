"""Tests for the Embedding domain value object."""

import math

import pytest

from domain.errors import EmptyEmbeddingError, NonFiniteEmbeddingValueError
from domain.value_objects.embedding import Embedding


def test_embedding_accepts_variable_dimension() -> None:
    """Embeddings must support different model dimensions."""
    embedding = Embedding.from_values([0.1, 0.2, 0.3])

    assert embedding.vector == (0.1, 0.2, 0.3)
    assert embedding.dimension == 3


def test_embedding_is_immutable() -> None:
    """Embedding vectors must not be mutable after creation."""
    embedding = Embedding.from_values([0.1, 0.2])

    with pytest.raises(AttributeError):
        embedding.vector = (0.3, 0.4)


def test_embedding_rejects_empty_vector() -> None:
    """An embedding must contain at least one value."""
    with pytest.raises(EmptyEmbeddingError):
        Embedding.from_values([])


@pytest.mark.parametrize(
    "value",
    [math.nan, math.inf, -math.inf],
)
def test_embedding_rejects_non_finite_values(value: float) -> None:
    """An embedding must contain only finite numeric values."""
    with pytest.raises(NonFiniteEmbeddingValueError):
        Embedding.from_values([0.1, value, 0.3])
