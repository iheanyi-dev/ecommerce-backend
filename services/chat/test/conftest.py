"""Shared PostgreSQL integration-test fixtures."""

from types import SimpleNamespace
from uuid import uuid4

import pytest_asyncio
from sqlalchemy import delete, or_


@pytest_asyncio.fixture
async def postgres_messages():
    from infrastructure.persistence.postgres.database import engine, session_factory
    from infrastructure.persistence.postgres.message_repository import (
        SQLAlchemyMessageRepository,
    )
    from infrastructure.persistence.postgres.models import MessageModel

    store_id = uuid4()
    conversation_id = uuid4()

    context = SimpleNamespace(
        repository=SQLAlchemyMessageRepository(session_factory),
        store_id=store_id,
        conversation_id=conversation_id,
    )

    try:
        yield context
    finally:
        try:
            async with session_factory() as session:
                async with session.begin():
                    await session.execute(
                        delete(MessageModel).where(
                            or_(
                                MessageModel.store_id == store_id,
                                MessageModel.conversation_id == conversation_id,
                            )
                        )
                    )
        finally:
            # Prevent pooled asyncpg connections from leaking across test loops.
            await engine.dispose()
