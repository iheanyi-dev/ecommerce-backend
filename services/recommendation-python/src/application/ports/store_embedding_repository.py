"""Persistence port for Store embedding representations."""

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


class StoreEmbeddingRepository(Protocol):
    """Persists Store embedding representations."""

    def upsert(
        self,
        store_id: EntityID,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> None:
        """Create or replace the Store embedding representation."""
        ...

    def delete(
        self,
        store_id: EntityID,
    ) -> None:
        """Delete the Store embedding representation."""
        ...

    def find_similar(
        self,
        embedding: Embedding,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        """Return Store IDs and similarity scores ordered by similarity."""
        ...
