"""Embedding domain errors."""

from .errors import (
    DomainError,
    EmptyEmbeddingError,
    EmptyEmbeddingModelVersionError,
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
    EmptyKnowledgeChunkContentError,
    EmptyStorageReferenceError,
    InvalidChunkIndexError,
    InvalidEntityIDError,
    NonFiniteEmbeddingValueError,
)

__all__ = [
    "DomainError",
    "EmptyEmbeddingError",
    "EmptyEmbeddingModelVersionError",
    "EmptyEntityDescriptionError",
    "EmptyEntityNameError",
    "EmptyKnowledgeChunkContentError",
    "EmptyStorageReferenceError",
    "InvalidChunkIndexError",
    "InvalidEntityIDError",
    "NonFiniteEmbeddingValueError",
]
