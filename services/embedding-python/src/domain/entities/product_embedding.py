"""Product embedding domain entity."""

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
class ProductEmbedding:
    """Persistable vector representation of a product."""

    id: UUID
    store_id: UUID
    name: str
    description: str
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion

    def __post_init__(self) -> None:
        EntityID(self.id, "product")
        EntityID(self.store_id, "store")
        EntityName(self.name, "product")
        EntityDescription(self.description, "product")
