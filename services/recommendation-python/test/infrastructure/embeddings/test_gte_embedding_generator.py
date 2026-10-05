"""Tests for the GTE embedding generator."""

from __future__ import annotations

from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest
import torch

from domain.value_objects import (
    EntityDescription,
    EntityName,
)
from infrastructure.embeddings.gte_embedding_generator import (
    GTEEmbeddingGenerator,
)
from infrastructure.embeddings.model_loader import LoadedEmbeddingModel
from infrastructure.embeddings.model_manifest import (
    EmbeddingModelManifest,
)


def _build_generator(
    *,
    dimension: int = 3,
) -> tuple[GTEEmbeddingGenerator, MagicMock, MagicMock, MagicMock]:
    """Build a generator around deterministic fake tokenizer/model objects."""
    tokenizer = MagicMock()

    input_ids = torch.tensor([[101, 200, 102]])
    attention_mask = torch.tensor([[1, 1, 1]])

    tokenizer.return_value = {
        "input_ids": input_ids,
        "attention_mask": attention_mask,
    }

    model = MagicMock()

    # Three-dimensional output makes the expected normalized vector obvious.
    model.return_value = SimpleNamespace(
        last_hidden_state=torch.tensor(
            [
                [
                    [3.0, 4.0, 0.0],
                    [100.0, 100.0, 100.0],
                    [200.0, 200.0, 200.0],
                ]
            ]
        )
    )

    loaded = LoadedEmbeddingModel(
        tokenizer=tokenizer,
        model=model,
        device=torch.device("cpu"),
    )

    manifest = EmbeddingModelManifest(
        model_id="test-model",
        revision="test-revision",
        embedding_dimension=dimension,
    )

    generator = GTEEmbeddingGenerator(
        loaded,
        manifest,
    )

    return generator, tokenizer, model, loaded


def test_generator_loads_model_once_at_construction() -> None:
    """The generator must not reload the model for every request."""
    generator, _, _, _ = _build_generator()

    generator.generate_store_query_embedding("coffee")
    generator.generate_product_query_embedding("shoes")



def test_store_embedding_uses_cls_pooling_and_l2_normalization() -> None:
    """Entity embeddings must use the normalized first-token representation."""
    generator, tokenizer, model, _ = _build_generator()

    embedding, version = generator.generate_store_embedding(
        EntityName("Coffee Shop", "Store"),
        EntityDescription("Specialty coffee", "Store"),
    )

    assert embedding.vector == pytest.approx(
        (0.6, 0.8, 0.0),
        abs=1e-6,
    )
    assert embedding.dimension == 3
    assert str(version) == "test-model@test-revision"

    tokenizer.assert_called_once_with(
        "Coffee Shop\nSpecialty coffee",
        padding=True,
        truncation=True,
        max_length=8192,
        return_tensors="pt",
    )

    model.assert_called_once()


def test_product_query_embedding_rejects_empty_query() -> None:
    """Similarity queries must contain meaningful text."""
    generator, _, _, _ = _build_generator()

    with pytest.raises(ValueError, match="query must not be empty"):
        generator.generate_product_query_embedding("   ")


def test_query_embedding_uses_same_normalized_representation() -> None:
    """Query embeddings must use the same CLS and normalization contract."""
    generator, tokenizer, _, _ = _build_generator()

    embedding = generator.generate_store_query_embedding("coffee")

    assert embedding.vector == pytest.approx(
        (0.6, 0.8, 0.0),
        abs=1e-6,
    )

    tokenizer.assert_called_once_with(
        "coffee",
        padding=True,
        truncation=True,
        max_length=8192,
        return_tensors="pt",
    )


def test_generator_rejects_manifest_dimension_mismatch() -> None:
    """Generated vectors must match the configured persistence dimension."""
    generator, _, _, _ = _build_generator(dimension=768)

    with pytest.raises(
        ValueError,
        match="does not match the model manifest",
    ):
        generator.generate_store_query_embedding("coffee")
