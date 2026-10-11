"""Qdrant persistence adapters for knowledge chunks."""

from .collections import (
    KNOWLEDGE_CHUNK_COLLECTION,
    ensure_knowledge_chunk_collection,
)
from .knowledge_chunk_repository import QdrantKnowledgeChunkRepository

__all__ = [
    "KNOWLEDGE_CHUNK_COLLECTION",
    "QdrantKnowledgeChunkRepository",
    "ensure_knowledge_chunk_collection",
]
