from datetime import datetime, timedelta, timezone
from uuid import uuid4

import pytest

from domain.entities.message import Message
from domain.value_objects.message_content import MessageContent
from domain.value_objects.message_role import MessageRole
from infrastructure.agent.conversation_history import (
    RepositoryConversationHistory,
)


def make_message(store_id, conversation_id, content, created_at):
    return Message(
        id=uuid4(),
        conversation_id=conversation_id,
        store_id=store_id,
        user_id=None,
        role=MessageRole.USER,
        content=MessageContent(content),
        created_at=created_at,
        updated_at=created_at,
    )


@pytest.mark.asyncio
async def test_history_excludes_current_message(postgres_messages):
    context = postgres_messages
    now = datetime.now(timezone.utc)

    previous = make_message(
        context.store_id,
        context.conversation_id,
        "Previous",
        now,
    )
    current = make_message(
        context.store_id,
        context.conversation_id,
        "Current",
        now + timedelta(seconds=1),
    )

    await context.repository.save(previous)
    await context.repository.save(current)

    history = RepositoryConversationHistory(context.repository)

    result = await history.get_history(
        store_id=context.store_id,
        conversation_id=context.conversation_id,
        exclude_message_id=current.id,
        limit=10,
    )

    assert result == [previous]


@pytest.mark.asyncio
async def test_history_returns_latest_prior_messages_in_order(
    postgres_messages,
):
    context = postgres_messages
    now = datetime.now(timezone.utc)

    previous_messages = [
        make_message(
            context.store_id,
            context.conversation_id,
            f"Prior {index}",
            now + timedelta(seconds=index),
        )
        for index in range(4)
    ]
    current = make_message(
        context.store_id,
        context.conversation_id,
        "Current",
        now + timedelta(seconds=4),
    )

    for message in [*previous_messages, current]:
        await context.repository.save(message)

    history = RepositoryConversationHistory(context.repository)

    result = await history.get_history(
        store_id=context.store_id,
        conversation_id=context.conversation_id,
        exclude_message_id=current.id,
        limit=2,
    )

    assert result == previous_messages[-2:]


@pytest.mark.asyncio
async def test_history_is_scoped_to_store_and_conversation(
    postgres_messages,
):
    context = postgres_messages
    now = datetime.now(timezone.utc)

    expected = make_message(
        context.store_id,
        context.conversation_id,
        "Expected",
        now,
    )
    different_store = make_message(
        uuid4(),
        context.conversation_id,
        "Different store",
        now + timedelta(seconds=1),
    )
    different_conversation = make_message(
        context.store_id,
        uuid4(),
        "Different conversation",
        now + timedelta(seconds=2),
    )

    for message in (expected, different_store, different_conversation):
        await context.repository.save(message)

    history = RepositoryConversationHistory(context.repository)

    result = await history.get_history(
        store_id=context.store_id,
        conversation_id=context.conversation_id,
        exclude_message_id=uuid4(),
        limit=10,
    )

    assert result == [expected]


@pytest.mark.asyncio
@pytest.mark.parametrize("limit", [0, -1])
async def test_history_rejects_non_positive_limit(postgres_messages, limit):
    history = RepositoryConversationHistory(postgres_messages.repository)

    with pytest.raises(ValueError):
        await history.get_history(
            store_id=postgres_messages.store_id,
            conversation_id=postgres_messages.conversation_id,
            exclude_message_id=uuid4(),
            limit=limit,
        )
