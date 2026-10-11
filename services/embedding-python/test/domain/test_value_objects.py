"""Focused tests for shared embedding domain value objects."""

import pytest

from domain.errors import (
    EmptyEmbeddingError,
    EmptyEmbeddingModelVersionError,
    NonFiniteEmbeddingValueError,
)
from domain.value_objects import Embedding, EmbeddingModelVersion


def test_embedding_is_immutable_and_reports_dimension() -> None:
    embedding = Embedding.from_values([0.1, 0.2])
    assert embedding.vector == (0.1, 0.2)
    assert embedding.dimension == 2


def test_embedding_rejects_empty_vector() -> None:
    with pytest.raises(EmptyEmbeddingError):
        Embedding.from_values([])


@pytest.mark.parametrize("value", [float("nan"), float("inf"), -float("inf")])
def test_embedding_rejects_non_finite_values(value: float) -> None:
    with pytest.raises(NonFiniteEmbeddingValueError):
        Embedding.from_values([value])


def test_model_version_rejects_empty_value() -> None:
    with pytest.raises(EmptyEmbeddingModelVersionError):
        EmbeddingModelVersion(" ")
