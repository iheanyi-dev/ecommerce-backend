"""Qdrant Product embedding repository implementation."""

from __future__ import annotations

from qdrant_client import QdrantClient

from application.ports import ProductEmbeddingRepository
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)

from .collections import PRODUCT_COLLECTION
from .embedding_repository import QdrantEmbeddingRepository


class QdrantProductEmbeddingRepository(ProductEmbeddingRepository):
    """Persists Product embeddings in the Qdrant Product collection."""

    def __init__(self, client: QdrantClient) -> None:
        self._repository = QdrantEmbeddingRepository(
            client=client,
            collection_name=PRODUCT_COLLECTION,
            entity_id_payload_key="product_id",
        )

    def upsert(
        self,
        product_id: EntityID,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> None:
        """Create or replace a Product embedding representation."""
        self._repository.upsert(
            entity_id=product_id,
            name=name,
            description=description,
            embedding=embedding,
            embedding_model_version=embedding_model_version,
        )

    def delete(
        self,
        product_id: EntityID,
    ) -> None:
        """Delete a Product embedding representation."""
        self._repository.delete(product_id)

    def find_similar(
        self,
        embedding: Embedding,
        limit: int,
    ):
        """Return Product IDs and similarity scores ordered by similarity."""
        return self._repository.find_similar(embedding, limit)
