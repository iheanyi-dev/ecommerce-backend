"""DTOs for creating or replacing a Store embedding."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from domain.value_objects import Embedding, EmbeddingModelVersion


@dataclass(frozen=True, slots=True)
class CreateStoreEmbeddingInput:
    """Primitive input received from the external service boundary."""

    store_id: UUID
    name: str
    description: str


@dataclass(frozen=True, slots=True)
class CreateStoreEmbeddingOutput:
    """Result returned after the Store embedding has been persisted."""

    store_id: UUID
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion
