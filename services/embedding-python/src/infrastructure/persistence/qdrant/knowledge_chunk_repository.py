"""Qdrant persistence and tenant-scoped retrieval for knowledge chunks."""

from __future__ import annotations

from collections.abc import Sequence
from uuid import UUID

from qdrant_client import QdrantClient, models

from application.ports.knowledge_chunk_repository import (
    KnowledgeChunkRepository,
)
from domain.entities.knowledge_chunk import KnowledgeChunk
from domain.value_objects import (
    ChunkIndex,
    Embedding,
    EmbeddingModelVersion,
    EntityID,
    KnowledgeChunkContent,
    StorageReference,
)

from .collections import KNOWLEDGE_CHUNK_COLLECTION


class QdrantKnowledgeChunkRepository(KnowledgeChunkRepository):
    """Persist document chunks and retrieve them within a single store."""

    def __init__(
        self,
        client: QdrantClient,
        collection_name: str = KNOWLEDGE_CHUNK_COLLECTION,
    ) -> None:
        if not collection_name.strip():
            raise ValueError("Qdrant collection name must not be empty")

        self._client = client
        self._collection_name = collection_name

    def replace_for_knowledge(
        self,
        knowledge_id: EntityID,
        chunks: Sequence[KnowledgeChunk],
    ) -> None:
        """Upsert replacement chunks before removing stale document points.

        Qdrant does not provide a transaction across these operations. Existing
        points are retained if the replacement upsert fails. If stale-point
        deletion fails after a successful upsert, duplicate old/new chunks may
        remain temporarily and a later replacement can clean them up.
        """
        document_id = str(knowledge_id.value)

        # An empty replacement means the document now has no indexed chunks.
        if not chunks:
            self.delete_by_knowledge_id(knowledge_id)
            return

        store_ids = {chunk.store_id for chunk in chunks}
        if len(store_ids) != 1:
            raise ValueError("All replacement chunks must belong to one store")

        if any(chunk.knowledge_id != knowledge_id.value for chunk in chunks):
            raise ValueError(
                "Every replacement chunk must match the requested knowledge ID"
            )

        # Read existing point IDs first so stale points can be removed only
        # after the new representation has been successfully written.
        existing_ids = self._get_point_ids_for_knowledge(document_id)

        points = [
            models.PointStruct(
                id=str(chunk.id),
                vector=list(chunk.embedding.vector),
                payload={
                    "knowledge_id": str(chunk.knowledge_id),
                    "store_id": str(chunk.store_id),
                    "storage_reference": chunk.storage_reference,
                    "content": chunk.content,
                    "chunk_index": chunk.chunk_index,
                    "embedding_model_version": str(
                        chunk.embedding_model_version
                    ),
                },
            )
            for chunk in chunks
        ]

        self._client.upsert(
            collection_name=self._collection_name,
            points=points,
            wait=True,
        )

        replacement_ids = {str(chunk.id) for chunk in chunks}
        stale_ids = existing_ids - replacement_ids

        if stale_ids:
            self._client.delete(
                collection_name=self._collection_name,
                points_selector=models.PointIdsList(points=sorted(stale_ids)),
                wait=True,
            )

    def delete_by_knowledge_id(self, knowledge_id: EntityID) -> None:
        """Delete every point for a document; repeated deletion is safe."""
        self._client.delete(
            collection_name=self._collection_name,
            points_selector=models.FilterSelector(
                filter=models.Filter(
                    must=[
                        models.FieldCondition(
                            key="knowledge_id",
                            match=models.MatchValue(
                                value=str(knowledge_id.value)
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
        store_id: EntityID,
    ) -> Sequence[KnowledgeChunk]:
        """Return nearest chunks, always filtering by the requested store."""
        if isinstance(top_k, bool) or not isinstance(top_k, int) or top_k <= 0:
            raise ValueError("top_k must be a positive integer")
        if isinstance(offset, bool) or not isinstance(offset, int) or offset < 0:
            raise ValueError("offset must be a non-negative integer")

        response = self._client.query_points(
            collection_name=self._collection_name,
            query=list(query_embedding.vector),
            limit=top_k,
            offset=offset,
            query_filter=models.Filter(
                must=[
                    models.FieldCondition(
                        key="store_id",
                        match=models.MatchValue(value=str(store_id.value)),
                    )
                ]
            ),
            with_payload=True,
            with_vectors=True,
        )

        results: list[KnowledgeChunk] = []
        for point in response.points:
            payload = point.payload
            vector = point.vector

            if not isinstance(payload, dict) or not isinstance(vector, list):
                raise ValueError(
                    f"Qdrant point {point.id} is missing payload or vector data"
                )

            results.append(
                KnowledgeChunk(
                    id=UUID(str(point.id)),
                    knowledge_id=UUID(str(payload["knowledge_id"])),
                    store_id=UUID(str(payload["store_id"])),
                    storage_reference=str(payload["storage_reference"]),
                    content=str(payload["content"]),
                    chunk_index=int(payload["chunk_index"]),
                    embedding=Embedding.from_values(vector),
                    embedding_model_version=EmbeddingModelVersion(
                        str(payload["embedding_model_version"])
                    ),
                )
            )

        return tuple(results)

    def _get_point_ids_for_knowledge(self, knowledge_id: str) -> set[str]:
        """Read all existing point IDs, including documents over 1,000 chunks."""
        point_ids: set[str] = set()
        next_offset = None

        while True:
            points, next_offset = self._client.scroll(
                collection_name=self._collection_name,
                scroll_filter=models.Filter(
                    must=[
                        models.FieldCondition(
                            key="knowledge_id",
                            match=models.MatchValue(value=knowledge_id),
                        )
                    ]
                ),
                limit=1000,
                offset=next_offset,
                with_payload=False,
                with_vectors=False,
            )
            point_ids.update(str(point.id) for point in points)

            if next_offset is None:
                return point_ids
