"""Conversation history adapter backed by message persistence."""

from uuid import UUID

from application.ports.message_repository import MessageRepository
from domain.entities.message import Message


class RepositoryConversationHistory:
    """Provide bounded, read-only history to the Agent."""

    def __init__(self, message_repository: MessageRepository) -> None:
        self._message_repository = message_repository

    async def get_history(
        self,
        *,
        store_id: UUID,
        conversation_id: UUID,
        exclude_message_id: UUID,
        limit: int,
    ) -> list[Message]:
        """Return the most recent prior messages in chronological order."""
        if limit < 1:
            raise ValueError("limit must be at least 1")

        # Fetch one extra record because the current message may be included.
        messages = await self._message_repository.get_by_conversation_id(
            store_id=store_id,
            conversation_id=conversation_id,
            limit=limit + 1,
        )

        prior_messages = [
            message
            for message in messages
            if message.id != exclude_message_id
        ]
        return prior_messages[-limit:]