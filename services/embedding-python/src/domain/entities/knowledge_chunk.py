"""Knowledge chunk domain entity."""

from dataclasses import dataclass
from uuid import UUID

from ..value_objects import (
    ChunkIndex,
    Embedding,
    EmbeddingModelVersion,
    EntityID,
    KnowledgeChunkContent,
    StorageReference,
)


@dataclass(slots=True)
class KnowledgeChunk:
    """A persistable, embedded chunk from a knowledge document."""

    id: UUID
    knowledge_id: UUID
    store_id: UUID
    storage_reference: str
    content: str
    chunk_index: int
    embedding: Embedding
    embedding_model_version: EmbeddingModelVersion

    def __post_init__(self) -> None:
        EntityID(self.id, "knowledge chunk")
        EntityID(self.knowledge_id, "knowledge")
        EntityID(self.store_id, "store")
        StorageReference(self.storage_reference)
        KnowledgeChunkContent(self.content)
        ChunkIndex(self.chunk_index)
