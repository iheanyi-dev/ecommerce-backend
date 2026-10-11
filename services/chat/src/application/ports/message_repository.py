"""Message persistence port."""

from typing import Protocol
from uuid import UUID
from domain.entities.message import Message


class MessageRepository(Protocol):
    """Persist messages independently of the storage implementation."""

    async def save(self, message: Message) -> None:
        """Persist a user or assistant message."""
        ...

    async def get_by_conversation_id(
        self,
        *,
        store_id: UUID,
        conversation_id: UUID,
        limit: int | None = None,
    ) -> list[Message]:
        """Retrieve messages belonging to a store conversation."""
        ...