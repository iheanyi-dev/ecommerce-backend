"""Dedicated Qdrant collection configuration."""

from qdrant_client import QdrantClient, models

KNOWLEDGE_CHUNK_COLLECTION = "knowledge_chunks"
STORE_EMBEDDING_COLLECTION = "store_embeddings"
PRODUCT_EMBEDDING_COLLECTION = "product_embeddings"


def _ensure_collection(
    client: QdrantClient,
    collection_name: str,
    vector_dimension: int,
) -> None:
    """Create a cosine-distance collection when it does not exist."""
    if (
        isinstance(vector_dimension, bool)
        or not isinstance(vector_dimension, int)
        or vector_dimension <= 0
    ):
        raise ValueError("Vector dimension must be a positive integer")

    if client.collection_exists(collection_name=collection_name):
        return

    client.create_collection(
        collection_name=collection_name,
        vectors_config=models.VectorParams(
            size=vector_dimension,
            distance=models.Distance.COSINE,
        ),
    )


def ensure_knowledge_chunk_collection(
    client: QdrantClient,
    vector_dimension: int,
) -> None:
    """Create the knowledge-chunk collection when absent."""
    _ensure_collection(
        client, KNOWLEDGE_CHUNK_COLLECTION, vector_dimension
    )


def ensure_store_embedding_collection(
    client: QdrantClient,
    vector_dimension: int,
) -> None:
    """Create the Store-embedding collection when absent."""
    _ensure_collection(
        client, STORE_EMBEDDING_COLLECTION, vector_dimension
    )


def ensure_product_embedding_collection(
    client: QdrantClient,
    vector_dimension: int,
) -> None:
    """Create the Product-embedding collection when absent."""
    _ensure_collection(
        client, PRODUCT_EMBEDDING_COLLECTION, vector_dimension
    )
