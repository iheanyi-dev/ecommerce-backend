"""Persistence port for Product embeddings."""

from typing import Protocol
from domain.entities import ProductEmbedding
from domain.value_objects import EntityID


class ProductEmbeddingRepository(Protocol):
    """Persists and deletes Product embedding representations."""

    def upsert(self, embedding: ProductEmbedding) -> None:
        """Create or replace a Product embedding."""
        ...

    def delete(self, product_id: EntityID) -> None:
        """Delete a Product embedding; repeated deletion must be safe."""
        ...

    def delete_by_store_id(self, store_id: EntityID) -> None:
        """Delete all Product embeddings associated with a Store."""
        ...

    def search(
        self, query_embedding, top_k: int, offset: int, store_id: EntityID | None = None
    ) -> list[ProductEmbedding]:
        """Return the closest Product embeddings, optionally scoped to a Store."""
        ...
