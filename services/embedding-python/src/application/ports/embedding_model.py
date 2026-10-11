"""Application port for the loaded embedding model."""

from typing import Protocol

from domain.value_objects import Embedding, EmbeddingModelVersion


class EmbeddingModel(Protocol):
    """Generates a vector and identifies the model revision used."""

    def generate(
        self, text: str
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        """Generate an embedding and return its exact model version."""
        ...
