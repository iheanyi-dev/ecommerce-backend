"""Qdrant persistence infrastructure."""

from .client import create_qdrant_client
from .collections import (
    PRODUCT_COLLECTION,
    STORE_COLLECTION,
    ensure_collection,
)
from .product_embedding_repository import QdrantProductEmbeddingRepository
from .store_embedding_repository import QdrantStoreEmbeddingRepository

__all__ = [
    "PRODUCT_COLLECTION",
    "STORE_COLLECTION",
    "QdrantProductEmbeddingRepository",
    "QdrantStoreEmbeddingRepository",
    "create_qdrant_client",
    "ensure_collection",
]
