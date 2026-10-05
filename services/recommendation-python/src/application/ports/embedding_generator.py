"""Port for generating embeddings using the configured embedding models."""

from __future__ import annotations

from typing import Protocol

from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityName,
)


class EmbeddingGenerator(Protocol):
    """Generates Store and Product embeddings.

    The implementation is responsible for loading and caching the
    appropriate embedding models. The application layer only knows
    that an embedding can be generated from validated domain values.
    """

    def generate_store_embedding(
        self,
        name: EntityName,
        description: EntityDescription,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        """Generate a Store embedding and return its model version."""
        ...

    def generate_store_query_embedding(
        self,
        query: str,
    ) -> Embedding:
        """Generate an embedding for a Store similarity query."""
        ...

    def generate_product_embedding(
        self,
        name: EntityName,
        description: EntityDescription,
    ) -> tuple[Embedding, EmbeddingModelVersion]:
        """Generate a Product embedding and return its model version."""
        ...

    def generate_product_query_embedding(
        self,
        query: str,
    ) -> Embedding:
        """Generate an embedding for a Product similarity query."""
        ...
