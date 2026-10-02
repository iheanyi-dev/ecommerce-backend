"""Recommendation domain errors."""

from .errors import (
    DomainError,
    EmptyEmbeddingError,
    EmptyEmbeddingModelVersionError,
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
    InvalidEntityIDError,
    NonFiniteEmbeddingValueError,
)

__all__ = [
    "DomainError",
    "EmptyEmbeddingError",
    "EmptyEmbeddingModelVersionError",
    "EmptyEntityDescriptionError",
    "EmptyEntityNameError",
    "InvalidEntityIDError",
    "NonFiniteEmbeddingValueError",
]
