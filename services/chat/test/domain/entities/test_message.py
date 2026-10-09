from datetime import UTC, datetime
from uuid import uuid4

from domain.entities.message import Message
from domain.value_objects.message_content import MessageContent
from domain.value_objects.message_role import MessageRole


def test_creates_message() -> None:
    message_id = uuid4()
    conversation_id = uuid4()
    store_id = uuid4()
    user_id = uuid4()
    created_at = datetime.now(UTC)

    message = Message(
        id=message_id,
        conversation_id=conversation_id,
        store_id=store_id,
        user_id=user_id,
        role=MessageRole.USER,
        content=MessageContent("Hello"),
        created_at=created_at,
        updated_at=created_at,
    )

    assert message.id == message_id
    assert message.conversation_id == conversation_id
    assert message.store_id == store_id
    assert message.user_id == user_id
    assert message.role is MessageRole.USER
    assert message.content.value == "Hello"
    assert message.created_at == created_at
    assert message.updated_at == created_at


def test_supports_anonymous_message() -> None:
    now = datetime.now(UTC)

    message = Message(
        id=uuid4(),
        conversation_id=uuid4(),
        store_id=uuid4(),
        user_id=None,
        role=MessageRole.USER,
        content=MessageContent("Hello"),
        created_at=now,
        updated_at=now,
    )

    assert message.user_id is None


def test_supports_assistant_message_for_authenticated_customer() -> None:
    now = datetime.now(UTC)
    user_id = uuid4()

    message = Message(
        id=uuid4(),
        conversation_id=uuid4(),
        store_id=uuid4(),
        user_id=user_id,
        role=MessageRole.ASSISTANT,
        content=MessageContent("How can I help?"),
        created_at=now,
        updated_at=now,
    )

    assert message.user_id == user_id
    assert message.role is MessageRole.ASSISTANT


def test_updates_content_and_timestamp() -> None:
    created_at = datetime.now(UTC)
    updated_at = datetime.fromtimestamp(
        created_at.timestamp() + 1,
        tz=UTC,
    )

    message = Message(
        id=uuid4(),
        conversation_id=uuid4(),
        store_id=uuid4(),
        user_id=None,
        role=MessageRole.ASSISTANT,
        content=MessageContent("Old response"),
        created_at=created_at,
        updated_at=created_at,
    )

    message.update_content(
        MessageContent("Updated response"),
        updated_at,
    )

    assert message.content.value == "Updated response"
    assert message.updated_at == updated_at
