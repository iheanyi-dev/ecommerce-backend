"""Agent execution port."""

from collections.abc import AsyncIterator
from typing import Protocol
from uuid import UUID


class Agent(Protocol):
    """Generate a response as a stream of text chunks."""

    async def run(
        self,
        store_id: UUID,
        conversation_id: UUID,
        current_message_id: UUID,
        message: str,
        system_prompt: str,
    ) -> AsyncIterator[str]:
        """Yield response text chunks."""
        ...
