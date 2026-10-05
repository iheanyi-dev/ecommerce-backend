"""Qdrant collection definitions and bootstrap helpers."""

from __future__ import annotations

from qdrant_client import QdrantClient, models


STORE_COLLECTION = "stores"
PRODUCT_COLLECTION = "products"


def ensure_collection(
    client: QdrantClient,
    collection_name: str,
    vector_dimension: int,
) -> None:
    """
    Create a Qdrant collection when it does not already exist.

    Collection creation is deliberately separate from repository operations.
    The embedding model determines the vector dimension, so that dimension is
    supplied by infrastructure configuration rather than being hardcoded here.
    """
    if not collection_name.strip():
        raise ValueError("Qdrant collection name must not be empty.")

    if vector_dimension <= 0:
        raise ValueError("Qdrant vector dimension must be greater than zero.")

    if client.collection_exists(collection_name=collection_name):
        return

    client.create_collection(
        collection_name=collection_name,
        vectors_config=models.VectorParams(
            size=vector_dimension,
            distance=models.Distance.COSINE,
        ),
    )
