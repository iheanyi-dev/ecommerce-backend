"""Read-only conversation history port for the agent."""

from typing import Protocol
from uuid import UUID

from domain.entities.message import Message


class ConversationHistory(Protocol):
    """Read prior messages without exposing persistence operations."""

    async def get_history(
        self,
        *,
        store_id: UUID,
        conversation_id: UUID,
        exclude_message_id: UUID,
        limit: int,
    ) -> list[Message]:
        """Return prior messages scoped to one store and conversation."""
        ...
