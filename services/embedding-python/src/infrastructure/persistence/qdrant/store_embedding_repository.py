"""Qdrant persistence for Store embeddings."""

from __future__ import annotations

from uuid import UUID

from qdrant_client import QdrantClient, models

from application.ports.store_embedding_repository import (
    StoreEmbeddingRepository,
)
from domain.entities.store_embedding import StoreEmbedding
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityID,
)

from .collections import STORE_EMBEDDING_COLLECTION


class QdrantStoreEmbeddingRepository(StoreEmbeddingRepository):
    """Persist and retrieve Store vectors."""

    def __init__(
        self,
        client: QdrantClient,
        collection_name: str = STORE_EMBEDDING_COLLECTION,
    ) -> None:
        if not collection_name.strip():
            raise ValueError("Qdrant collection name must not be empty")

        self._client = client
        self._collection_name = collection_name

    def upsert(self, embedding: StoreEmbedding) -> None:
        """Create or replace a Store vector using its Store ID as point ID."""
        self._client.upsert(
            collection_name=self._collection_name,
            points=[
                models.PointStruct(
                    id=str(embedding.id),
                    vector=list(embedding.embedding.vector),
                    payload={
                        "name": embedding.name,
                        "description": embedding.description,
                        "embedding_model_version": str(
                            embedding.embedding_model_version
                        ),
                    },
                )
            ],
            wait=True,
        )

    def delete(self, store_id: EntityID) -> None:
        """Delete a Store vector; deleting an absent point is safe."""
        self._client.delete(
            collection_name=self._collection_name,
            points_selector=models.PointIdsList(
                points=[str(store_id.value)]
            ),
            wait=True,
        )

    def search(
        self,
        query_embedding: Embedding,
        top_k: int,
        offset: int,
    ) -> list[StoreEmbedding]:
        """Search Store vectors by similarity with pagination."""
        _validate_pagination(top_k, offset)

        response = self._client.query_points(
            collection_name=self._collection_name,
            query=list(query_embedding.vector),
            limit=top_k,
            offset=offset,
            with_payload=True,
            with_vectors=True,
        )

        results: list[StoreEmbedding] = []

        for point in response.points:
            payload = point.payload
            vector = point.vector

            if not isinstance(payload, dict) or not isinstance(vector, list):
                raise ValueError(
                    f"Qdrant point {point.id} is missing payload or vector data"
                )

            results.append(
                StoreEmbedding(
                    id=UUID(str(point.id)),
                    name=str(payload["name"]),
                    description=str(payload["description"]),
                    embedding=Embedding.from_values(vector),
                    embedding_model_version=EmbeddingModelVersion(
                        str(payload["embedding_model_version"])
                    ),
                )
            )

        return results


def _validate_pagination(top_k: int, offset: int) -> None:
    if isinstance(top_k, bool) or not isinstance(top_k, int) or top_k <= 0:
        raise ValueError("top_k must be a positive integer")

    if isinstance(offset, bool) or not isinstance(offset, int) or offset < 0:
        raise ValueError("offset must be a non-negative integer")
