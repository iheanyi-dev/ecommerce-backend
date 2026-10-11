"""SQLAlchemy implementation of the message repository."""

from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from application.ports.message_repository import MessageRepository
from domain.entities.message import Message
from domain.value_objects.message_content import MessageContent
from domain.value_objects.message_role import MessageRole
from infrastructure.persistence.postgres.models import MessageModel


class SQLAlchemyMessageRepository(MessageRepository):
    """Persist and retrieve messages using PostgreSQL."""

    def __init__(
        self,
        session_factory: async_sessionmaker[AsyncSession],
    ) -> None:
        self._session_factory = session_factory

    async def save(self, message: Message) -> None:
        """Insert a message and commit the transaction."""
        async with self._session_factory() as session:
            async with session.begin():
                session.add(
                    MessageModel(
                        id=message.id,
                        conversation_id=message.conversation_id,
                        store_id=message.store_id,
                        user_id=message.user_id,
                        role=message.role.value,
                        content=message.content.value,
                        created_at=message.created_at,
                        updated_at=message.updated_at,
                    )
                )

    async def get_by_conversation_id(
        self,
        *,
        store_id: UUID,
        conversation_id: UUID,
        limit: int | None = None,
    ) -> list[Message]:
        """Retrieve conversation messages in chronological order.

        When a limit is supplied, retrieve the most recent messages while
        returning them in chronological order.
        """
        statement = (
            select(MessageModel)
            .where(
                MessageModel.store_id == store_id,
                MessageModel.conversation_id == conversation_id,
            )
            .order_by(
                MessageModel.created_at.desc(),
                MessageModel.id.desc(),
            )
        )

        if limit is not None:
            if limit < 1:
                raise ValueError("limit must be at least 1")
            statement = statement.limit(limit)

        async with self._session_factory() as session:
            result = await session.scalars(statement)
            records = list(result.all())

        records.reverse()
        return [
            Message(
                id=record.id,
                conversation_id=record.conversation_id,
                store_id=record.store_id,
                user_id=record.user_id,
                role=MessageRole.from_value(record.role),
                content=MessageContent(record.content),
                created_at=record.created_at,
                updated_at=record.updated_at,
            )
            for record in records
        ]