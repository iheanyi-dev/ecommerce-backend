from datetime import datetime, timedelta, timezone
from uuid import uuid4

import pytest

from domain.entities.message import Message
from domain.value_objects.message_content import MessageContent
from domain.value_objects.message_role import MessageRole


def make_message(
    store_id,
    conversation_id,
    content,
    created_at,
    *,
    user_id=None,
    role=MessageRole.USER,
):
    return Message(
        id=uuid4(),
        conversation_id=conversation_id,
        store_id=store_id,
        user_id=user_id,
        role=role,
        content=MessageContent(content),
        created_at=created_at,
        updated_at=created_at,
    )


@pytest.mark.asyncio
async def test_save_and_get_messages(postgres_messages):
    context = postgres_messages
    now = datetime.now(timezone.utc)

    first = make_message(
        context.store_id,
        context.conversation_id,
        "First message",
        now,
    )
    second = make_message(
        context.store_id,
        context.conversation_id,
        "Second message",
        now + timedelta(seconds=1),
        role=MessageRole.ASSISTANT,
    )

    await context.repository.save(first)
    await context.repository.save(second)

    result = await context.repository.get_by_conversation_id(
        store_id=context.store_id,
        conversation_id=context.conversation_id,
    )

    assert result == [first, second]


@pytest.mark.asyncio
async def test_limit_returns_latest_messages_in_chronological_order(
    postgres_messages,
):
    context = postgres_messages
    now = datetime.now(timezone.utc)
    messages = [
        make_message(
            context.store_id,
            context.conversation_id,
            f"Message {index}",
            now + timedelta(seconds=index),
        )
        for index in range(4)
    ]

    for message in messages:
        await context.repository.save(message)

    result = await context.repository.get_by_conversation_id(
        store_id=context.store_id,
        conversation_id=context.conversation_id,
        limit=2,
    )

    assert result == messages[-2:]


@pytest.mark.asyncio
async def test_get_messages_is_scoped_to_store_and_conversation(
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

    result = await context.repository.get_by_conversation_id(
        store_id=context.store_id,
        conversation_id=context.conversation_id,
    )

    assert result == [expected]


@pytest.mark.asyncio
@pytest.mark.parametrize("limit", [0, -1])
async def test_non_positive_limit_is_rejected(postgres_messages, limit):
    with pytest.raises(ValueError):
        await postgres_messages.repository.get_by_conversation_id(
            store_id=postgres_messages.store_id,
            conversation_id=postgres_messages.conversation_id,
            limit=limit,
        )
