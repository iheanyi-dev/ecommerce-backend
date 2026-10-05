"""Qdrant Store embedding repository implementation."""

from __future__ import annotations

from qdrant_client import QdrantClient

from application.ports import StoreEmbeddingRepository
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)

from .collections import STORE_COLLECTION
from .embedding_repository import QdrantEmbeddingRepository


class QdrantStoreEmbeddingRepository(StoreEmbeddingRepository):
    """Persists Store embeddings in the Qdrant Store collection."""

    def __init__(self, client: QdrantClient) -> None:
        self._repository = QdrantEmbeddingRepository(
            client=client,
            collection_name=STORE_COLLECTION,
            entity_id_payload_key="store_id",
        )

    def upsert(
        self,
        store_id: EntityID,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> None:
        """Create or replace a Store embedding representation."""
        self._repository.upsert(
            entity_id=store_id,
            name=name,
            description=description,
            embedding=embedding,
            embedding_model_version=embedding_model_version,
        )

    def delete(
        self,
        store_id: EntityID,
    ) -> None:
        """Delete a Store embedding representation."""
        self._repository.delete(store_id)

    def find_similar(
        self,
        embedding: Embedding,
        limit: int,
    ):
        """Return Store IDs and similarity scores ordered by similarity."""
        return self._repository.find_similar(embedding, limit)
