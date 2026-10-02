"""Tests for the EmbeddingModelVersion domain value object."""

import pytest

from domain.errors import EmptyEmbeddingModelVersionError
from domain.value_objects.embedding_model_version import EmbeddingModelVersion


def test_embedding_model_version_accepts_valid_value() -> None:
    """A model version must preserve its supplied value."""
    version = EmbeddingModelVersion("store-v1")

    assert version.value == "store-v1"
    assert str(version) == "store-v1"


def test_embedding_model_version_rejects_empty_value() -> None:
    """A model version cannot be empty."""
    with pytest.raises(EmptyEmbeddingModelVersionError):
        EmbeddingModelVersion("   ")
