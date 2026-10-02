"""Domain entity representing a Product for recommendation purposes."""

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
class Product:
    """
    Recommendation-owned representation of a Product.

    The Recommendation Service keeps only the Product information required
    to generate and serve product recommendations. Product business concerns
    such as ownership, inventory, pricing, and orders remain outside this
    domain.
    """

    id: UUID
    name: str
    description: str
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion

    def __post_init__(self) -> None:
        """Validate the Product's domain invariants."""
        if not isinstance(self.id, UUID):
            raise InvalidEntityIDError("product")

        if not self.name.strip():
            raise EmptyEntityNameError("product")

        if not self.description.strip():
            raise EmptyEntityDescriptionError("product")
