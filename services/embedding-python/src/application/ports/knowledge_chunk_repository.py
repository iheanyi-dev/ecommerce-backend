"""Port for persistence and semantic retrieval of knowledge chunks."""

from typing import Protocol, Sequence

from domain.entities.knowledge_chunk import KnowledgeChunk
from domain.value_objects.embedding import Embedding
from domain.value_objects.entity_id import EntityID


class KnowledgeChunkRepository(Protocol):
    """Define persistence operations required by knowledge use cases."""

    def replace_for_knowledge(
        self,
        knowledge_id: EntityID,
        chunks: Sequence[KnowledgeChunk],
    ) -> None:
        """Atomically replace every indexed chunk for one knowledge document."""
        ...

    def delete_by_knowledge_id(self, knowledge_id: EntityID) -> None:
        """Delete all chunks for a document; repeated deletion must be safe."""
        ...

    def search(
        self,
        query_embedding: Embedding,
        top_k: int,
        offset: int,
        store_id: EntityID,
    ) -> Sequence[KnowledgeChunk]:
        """Return ranked chunks restricted to the requested store."""
        ...
