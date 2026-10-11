"""DTOs for creating or replacing a Store embedding."""

from dataclasses import dataclass
from uuid import UUID

from domain.value_objects import Embedding, EmbeddingModelVersion


@dataclass(frozen=True, slots=True)
class CreateStoreEmbeddingInput:
    """Store embedding request received from an external boundary."""

    store_id: UUID
    name: str
    description: str


@dataclass(frozen=True, slots=True)
class CreateStoreEmbeddingOutput:
    """Result after persisting a Store embedding."""

    store_id: UUID
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion
