"""Tests for the immutable embedding model manifest."""

from __future__ import annotations

import pytest

from infrastructure.embeddings.model_manifest import (
    GTE_MULTILINGUAL_BASE,
    EmbeddingModelManifest,
)


def test_gte_multilingual_base_manifest_is_locked() -> None:
    """The configured GTE model must expose its immutable contract."""
    assert GTE_MULTILINGUAL_BASE.model_id == "Alibaba-NLP/gte-multilingual-base"
    assert (
        GTE_MULTILINGUAL_BASE.revision
        == "9bbca17d9273fd0d03d5725c7a4b0f6b45142062"
    )
    assert GTE_MULTILINGUAL_BASE.embedding_dimension == 768


@pytest.mark.parametrize(
    ("model_id", "revision", "dimension"),
    [
        ("", "revision", 768),
        ("model", "", 768),
        ("model", "revision", 0),
    ],
)
def test_manifest_rejects_invalid_configuration(
    model_id: str,
    revision: str,
    dimension: int,
) -> None:
    """Invalid model contracts must fail during construction."""
    with pytest.raises(ValueError):
        EmbeddingModelManifest(
            model_id=model_id,
            revision=revision,
            embedding_dimension=dimension,
        )
