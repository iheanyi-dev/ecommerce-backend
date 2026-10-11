"""Port for retrieving store-scoped knowledge chunks."""

from dataclasses import dataclass
from typing import Protocol
from uuid import UUID


@dataclass(frozen=True, slots=True)
class KnowledgeChunk:
    """A relevant passage returned by the knowledge retrieval service."""

    content: str
    source_filename: str
    score: float


class KnowledgeRetrieval(Protocol):
    """Retrieve relevant knowledge for a store."""

    async def retrieve_chunks(
        self,
        *,
        store_id: UUID,
        query: str,
    ) -> list[KnowledgeChunk]:
        """Return relevant chunks for the supplied store and query."""
        ...
