"""Message persistence port."""

from typing import Protocol

from domain.entities.message import Message


class MessageRepository(Protocol):
    """Persist messages independently of the storage implementation."""

    async def save(self, message: Message) -> None:
        """Persist a user or assistant message."""
        ...
