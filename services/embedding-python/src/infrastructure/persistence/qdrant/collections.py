"""Dedicated Qdrant collection configuration for knowledge chunks."""

from qdrant_client import QdrantClient, models

KNOWLEDGE_CHUNK_COLLECTION = "knowledge_chunks"


def ensure_knowledge_chunk_collection(
    client: QdrantClient,
    vector_dimension: int,
) -> None:
    """Create the dedicated knowledge collection when it does not exist."""
    if (
        isinstance(vector_dimension, bool)
        or not isinstance(vector_dimension, int)
        or vector_dimension <= 0
    ):
        raise ValueError("Vector dimension must be a positive integer")

    if client.collection_exists(
        collection_name=KNOWLEDGE_CHUNK_COLLECTION
    ):
        return

    client.create_collection(
        collection_name=KNOWLEDGE_CHUNK_COLLECTION,
        vectors_config=models.VectorParams(
            size=vector_dimension,
            distance=models.Distance.COSINE,
        ),
    )
