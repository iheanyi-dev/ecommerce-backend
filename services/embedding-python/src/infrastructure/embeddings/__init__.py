"""Embedding model infrastructure adapters."""

from .gte_embedding_model import GTEEmbeddingModel
from .model_manifest import (
    GTE_MULTILINGUAL_BASE,
    EmbeddingModelManifest,
)

__all__ = [
    "GTEEmbeddingModel",
    "GTE_MULTILINGUAL_BASE",
    "EmbeddingModelManifest",
]
