"""DTOs for creating or replacing a Product embedding."""

from dataclasses import dataclass
from uuid import UUID

from domain.value_objects import Embedding, EmbeddingModelVersion


@dataclass(frozen=True, slots=True)
class CreateProductEmbeddingInput:
    """Product embedding request received from an external boundary."""

    product_id: UUID
    store_id: UUID
    name: str
    description: str


@dataclass(frozen=True, slots=True)
class CreateProductEmbeddingOutput:
    """Result after persisting a Product embedding."""

    product_id: UUID
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion
