"""Request and response contracts for knowledge retrieval."""

from dataclasses import dataclass
from uuid import UUID

from domain.entities.knowledge_chunk import KnowledgeChunk


@dataclass(frozen=True, slots=True)
class RetrieveKnowledgeInput:
    """Describe a tenant-scoped semantic search request.

    Defaults keep the common call simple while still allowing callers to
    request a different result count or page.
    """

    store_id: UUID
    query: str
    top_k: int = 10
    offset: int = 0


@dataclass(frozen=True, slots=True)
class RetrieveKnowledgeOutput:
    """Return the matching chunks without exposing repository internals."""

    store_id: UUID
    chunks: tuple[KnowledgeChunk, ...]
