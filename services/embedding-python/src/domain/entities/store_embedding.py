"""Store embedding domain entity."""

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
class StoreEmbedding:
    """Persistable vector representation of a store."""

    id: UUID
    name: str
    description: str
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion

    def __post_init__(self) -> None:
        EntityID(self.id, "store")
        EntityName(self.name, "store")
        EntityDescription(self.description, "store")
