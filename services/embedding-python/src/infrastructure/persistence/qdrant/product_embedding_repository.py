from __future__ import annotations
from application.dto.scored_product_embedding import ScoredProductEmbedding
"""Qdrant persistence for Product embeddings."""


from uuid import UUID

from qdrant_client import QdrantClient, models

from application.ports.product_embedding_repository import (
    ProductEmbeddingRepository,
)
from domain.entities.product_embedding import ProductEmbedding
from domain.value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityID,
)

from .collections import PRODUCT_EMBEDDING_COLLECTION
from .store_embedding_repository import _validate_pagination


class QdrantProductEmbeddingRepository(ProductEmbeddingRepository):
    """Persist and retrieve Product vectors, optionally scoped by Store."""

    def __init__(
        self,
        client: QdrantClient,
        collection_name: str = PRODUCT_EMBEDDING_COLLECTION,
    ) -> None:
        if not collection_name.strip():
            raise ValueError("Qdrant collection name must not be empty")

        self._client = client
        self._collection_name = collection_name

    def upsert(self, embedding: ProductEmbedding) -> None:
        """Create or replace a Product vector using its Product ID."""
        self._client.upsert(
            collection_name=self._collection_name,
            points=[
                models.PointStruct(
                    id=str(embedding.id),
                    vector=list(embedding.embedding.vector),
                    payload={
                        "store_id": str(embedding.store_id),
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

    def delete(self, product_id: EntityID) -> None:
        """Delete a Product vector; deleting an absent point is safe."""
        self._client.delete(
            collection_name=self._collection_name,
            points_selector=models.PointIdsList(
                points=[str(product_id.value)]
            ),
            wait=True,
        )

    def delete_by_store_id(self, store_id: EntityID) -> None:
        """Delete every Product vector belonging to a Store."""
        self._client.delete(
            collection_name=self._collection_name,
            points_selector=models.FilterSelector(
                filter=models.Filter(
                    must=[
                        models.FieldCondition(
                            key="store_id",
                            match=models.MatchValue(
                                value=str(store_id.value)
                            ),
                        )
                    ]
                )
            ),
            wait=True,
        )

    def search(
        self,
        query_embedding: Embedding,
        top_k: int,
        offset: int,
        store_id: EntityID | None = None,
    ) -> list[ScoredProductEmbedding]:
        """Search Products by similarity, optionally filtering by Store."""
        _validate_pagination(top_k, offset)

        query_filter = None
        if store_id is not None:
            query_filter = models.Filter(
                must=[
                    models.FieldCondition(
                        key="store_id",
                        match=models.MatchValue(
                            value=str(store_id.value)
                        ),
                    )
                ]
            )

        response = self._client.query_points(
            collection_name=self._collection_name,
            query=list(query_embedding.vector),
            limit=top_k,
            offset=offset,
            query_filter=query_filter,
            with_payload=True,
            with_vectors=True,
        )

        results: list[ScoredProductEmbedding] = []

        for point in response.points:
            payload = point.payload
            vector = point.vector

            if not isinstance(payload, dict) or not isinstance(vector, list):
                raise ValueError(
                    f"Qdrant point {point.id} is missing payload or vector data"
                )

            results.append(
                ScoredProductEmbedding(
                    embedding=ProductEmbedding(
                    id=UUID(str(point.id)),
                    store_id=UUID(str(payload["store_id"])),
                    name=str(payload["name"]),
                    description=str(payload["description"]),
                    embedding=Embedding.from_values(vector),
                    embedding_model_version=EmbeddingModelVersion(
                        str(payload["embedding_model_version"])
                    ),
                    ),
                    score=float(point.score),
                )
            )

        return results
