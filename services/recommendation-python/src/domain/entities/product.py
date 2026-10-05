"""Recommendation domain Product entity."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from ..value_objects import (
    Embedding,
    EmbeddingModelVersion,
    EntityDescription,
    EntityID,
    EntityName,
)


@dataclass(slots=True)
class Product:
    """Product representation required by the recommendation domain."""

    id: UUID
    name: str
    description: str
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion

    def __post_init__(self) -> None:
        EntityID(self.id, "product")
        EntityName(self.name, "product")
        EntityDescription(self.description, "product")
