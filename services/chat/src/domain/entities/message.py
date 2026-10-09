"""Message domain entity."""

from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

from domain.value_objects.message_content import MessageContent
from domain.value_objects.message_role import MessageRole


@dataclass(slots=True)
class Message:
    """A message belonging to a store/customer conversation."""

    id: UUID
    conversation_id: UUID
    store_id: UUID
    user_id: UUID | None
    role: MessageRole
    content: MessageContent
    created_at: datetime
    updated_at: datetime

    def update_content(
        self,
        content: MessageContent,
        updated_at: datetime,
    ) -> None:
        """Replace the message content and update its timestamp."""
        self.content = content
        self.updated_at = updated_at
