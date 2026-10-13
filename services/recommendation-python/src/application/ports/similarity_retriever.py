"""Application port for semantic candidate retrieval."""

from __future__ import annotations

from typing import Protocol
from uuid import UUID


class SimilarityRetriever(Protocol):
    """Retrieve ranked candidates from the Embedding service."""

    def find_similar_products(
        self,
        query: str,
        top_k: int,
        offset: int,
    ) -> list[tuple[UUID, float]]:
        """Return product IDs and similarity scores."""

    def find_similar_stores(
        self,
        query: str,
        top_k: int,
        offset: int,
    ) -> list[tuple[UUID, float]]:
        """Return store IDs and similarity scores."""
