"""Pinned configuration for the GTE multilingual embedding model."""

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class EmbeddingModelManifest:
    """Identifies the exact model revision and expected vector dimension."""

    model_id: str
    revision: str
    embedding_dimension: int

    def __post_init__(self) -> None:
        if not self.model_id.strip():
            raise ValueError("model_id must not be empty")
        if not self.revision.strip():
            raise ValueError("revision must not be empty")
        if (
            isinstance(self.embedding_dimension, bool)
            or not isinstance(self.embedding_dimension, int)
            or self.embedding_dimension <= 0
        ):
            raise ValueError("embedding_dimension must be a positive integer")


GTE_MULTILINGUAL_BASE = EmbeddingModelManifest(
    model_id="Alibaba-NLP/gte-multilingual-base",
    revision="9bbca17d9273fd0d03d5725c7a4b0f6b45142062",
    embedding_dimension=768,
)
