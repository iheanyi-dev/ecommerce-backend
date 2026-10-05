"""DTOs for creating or replacing a Product embedding."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from domain.value_objects import Embedding, EmbeddingModelVersion


@dataclass(frozen=True, slots=True)
class CreateProductEmbeddingInput:
    """Primitive input received from the external service boundary."""

    product_id: UUID
    name: str
    description: str


@dataclass(frozen=True, slots=True)
class CreateProductEmbeddingOutput:
    """Result returned after the Product embedding has been persisted."""

    product_id: UUID
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion
