"""Persistence port for Store embeddings."""

from typing import Protocol
from domain.entities import StoreEmbedding
from domain.value_objects import EntityID


class StoreEmbeddingRepository(Protocol):
    """Persists and deletes Store embedding representations."""

    def upsert(self, embedding: StoreEmbedding) -> None:
        """Create or replace a Store embedding."""
        ...

    def delete(self, store_id: EntityID) -> None:
        """Delete a Store embedding; repeated deletion must be safe."""
        ...

    def search(self, query_embedding, top_k: int, offset: int) -> list[StoreEmbedding]:
        """Return the closest Store embeddings using pagination."""
        ...
