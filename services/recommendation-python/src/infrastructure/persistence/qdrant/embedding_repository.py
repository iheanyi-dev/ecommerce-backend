"""Shared Qdrant implementation for entity embedding persistence."""

from __future__ import annotations

from uuid import UUID

from qdrant_client import QdrantClient, models

from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)


class QdrantEmbeddingRepository:
    """
    Shared infrastructure implementation for entity embeddings.

    Store and Product repositories expose separate application ports, while
    their Qdrant persistence behavior is identical apart from the collection
    name and entity-specific payload identifier.
    """

    def __init__(
        self,
        client: QdrantClient,
        collection_name: str,
        entity_id_payload_key: str,
    ) -> None:
        if not collection_name.strip():
            raise ValueError("Qdrant collection name must not be empty.")

        if not entity_id_payload_key.strip():
            raise ValueError("Entity ID payload key must not be empty.")

        self._client = client
        self._collection_name = collection_name
        self._entity_id_payload_key = entity_id_payload_key

    def upsert(
        self,
        entity_id: EntityID,
        name: EntityName,
        description: EntityDescription,
        embedding: Embedding,
        embedding_model_version: EmbeddingModelVersion,
    ) -> None:
        """Create or replace an entity's embedding representation."""
        point_id = str(entity_id.value)

        point = models.PointStruct(
            id=point_id,
            vector=list(embedding.vector),
            payload={
                self._entity_id_payload_key: point_id,
                "name": str(name),
                "description": str(description),
                "embedding_model_version": str(embedding_model_version),
            },
        )

        self._client.upsert(
            collection_name=self._collection_name,
            points=[point],
            wait=True,
        )

    def delete(
        self,
        entity_id: EntityID,
    ) -> None:
        """Delete an entity's embedding representation."""
        self._client.delete(
            collection_name=self._collection_name,
            points_selector=models.PointIdsList(
                points=[str(entity_id.value)],
            ),
            wait=True,
        )

    def find_similar(
        self,
        embedding: Embedding,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        """
        Find the nearest stored entity embeddings.

        Qdrant returns points ordered by similarity score, so this method
        preserves that ordering for the application layer.
        """
        if limit <= 0:
            raise ValueError("Similarity search limit must be greater than zero.")

        response = self._client.query_points(
            collection_name=self._collection_name,
            query=list(embedding.vector),
            limit=limit,
        )

        results: list[tuple[UUID, float]] = []

        for point in response.points:
            results.append(
                (
                    UUID(str(point.id)),
                    float(point.score),
                )
            )

        return results
