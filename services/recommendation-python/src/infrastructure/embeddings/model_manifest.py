"""Immutable configuration for the Recommendation Service embedding model."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class EmbeddingModelManifest:
    """Describes the exact embedding model artifact required at runtime."""

    model_id: str
    revision: str
    embedding_dimension: int

    def __post_init__(self) -> None:
        """Reject incomplete model configuration."""
        if not self.model_id.strip():
            raise ValueError("Embedding model ID must not be empty.")

        if not self.revision.strip():
            raise ValueError("Embedding model revision must not be empty.")

        if self.embedding_dimension <= 0:
            raise ValueError(
                "Embedding model dimension must be greater than zero."
            )




GTE_MULTILINGUAL_BASE = EmbeddingModelManifest(
    model_id="Alibaba-NLP/gte-multilingual-base",
    revision="9bbca17d9273fd0d03d5725c7a4b0f6b45142062",
    embedding_dimension=768,
)
