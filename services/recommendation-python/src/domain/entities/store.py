"""Domain entity representing a Store for recommendation purposes."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from ..errors import (
    EmptyEntityDescriptionError,
    EmptyEntityNameError,
    InvalidEntityIDError,
)
from ..value_objects.embedding import Embedding
from ..value_objects.embedding_model_version import EmbeddingModelVersion


@dataclass(slots=True)
class Store:
    """
    Recommendation-owned representation of a Store.

    The Recommendation Service keeps only the Store information required
    to generate and serve store recommendations. Store business concerns
    such as ownership, status, plans, billing, and products remain owned
    by the Store Service.
    """

    id: UUID
    name: str
    description: str
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion

    def __post_init__(self) -> None:
        """Validate the Store's domain invariants."""
        if not isinstance(self.id, UUID):
            raise InvalidEntityIDError("store")

        if not self.name.strip():
            raise EmptyEntityNameError("store")

        if not self.description.strip():
            raise EmptyEntityDescriptionError("store")
