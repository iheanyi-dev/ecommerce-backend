"""Input contract for deleting all indexed chunks belonging to knowledge."""

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class DeleteKnowledgeInput:
    """Identify the knowledge document whose indexed chunks must be removed."""

    knowledge_id: UUID
