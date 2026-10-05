"""Persistence port for Product embedding representations."""

from __future__ import annotations

from typing import Protocol
from uuid import UUID

from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)


class ProductEmbeddingRepository(Protocol):
    """Persists Product embedding representations."""

    def upsert(
        self,
        product_id: EntityID,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> None:
        """Create or replace the Product embedding representation."""
        ...

    def delete(
        self,
        product_id: EntityID,
    ) -> None:
        """Delete the Product embedding representation."""
        ...

    def find_similar(
        self,
        embedding: Embedding,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        """Return Product IDs and similarity scores ordered by similarity."""
        ...
