"""Tests for the StreamMessage application use case."""

from collections.abc import AsyncIterator
from datetime import UTC, datetime
from uuid import UUID, uuid4

import pytest

from application.ports.logger import Logger
from application.ports.message_repository import MessageRepository
from application.ports.metrics import Metrics
from application.use_cases.stream_message import StreamMessage
from domain.entities.message import Message
from domain.value_objects.message_role import MessageRole


class StubMessageRepository:
    """In-memory message repository for use-case tests."""

    def __init__(self) -> None:
        self.messages: list[Message] = []
        self.fail_on_save = False

    async def save(self, message: Message) -> None:
        if self.fail_on_save:
            raise RuntimeError("message persistence failed")
        self.messages.append(message)


class StubAgent:
    """Agent stub that streams configured response chunks."""

    def __init__(self, chunks: list[str] | None = None) -> None:
        self.chunks = chunks if chunks is not None else ["Hello", " there!"]
        self.calls: list[dict[str, object]] = []
        self.error: Exception | None = None

    async def run(
        self,
        store_id: UUID,
        conversation_id: UUID,
        current_message_id: UUID,
        message: str,
        system_prompt: str,
    ) -> AsyncIterator[str]:
        self.calls.append(
            {
                "store_id": store_id,
                "conversation_id": conversation_id,
                "current_message_id": current_message_id,
                "message": message,
                "system_prompt": system_prompt,
            }
        )

        for chunk in self.chunks:
            yield chunk

        if self.error is not None:
            raise self.error


class StubLogger:
    """Captures structured application log events."""

    def __init__(self) -> None:
        self.events: list[tuple[str, str, dict[str, object]]] = []

    def info(self, event: str, **fields: object) -> None:
        self.events.append(("info", event, fields))

    def warning(self, event: str, **fields: object) -> None:
        self.events.append(("warning", event, fields))

    def error(self, event: str, **fields: object) -> None:
        self.events.append(("error", event, fields))


class StubMetrics:
    """Captures metric increments and observations."""

    def __init__(self) -> None:
        self.counters: dict[str, int] = {}
        self.observations: dict[str, list[float]] = {}

    def increment(self, name: str, value: int = 1) -> None:
        self.counters[name] = self.counters.get(name, 0) + value

    def observe(self, name: str, value: float) -> None:
        self.observations.setdefault(name, []).append(value)


def build_use_case(
    *,
    chunks: list[str] | None = None,
) -> tuple[
    StreamMessage,
    StubMessageRepository,
    StubAgent,
    StubLogger,
    StubMetrics,
]:
    repository = StubMessageRepository()
    agent = StubAgent(chunks)
    logger = StubLogger()
    metrics = StubMetrics()

    use_case = StreamMessage(
        message_repository=repository,
        agent=agent,
        logger=logger,
        metrics=metrics,
    )
    return use_case, repository, agent, logger, metrics


async def collect(stream: AsyncIterator[str]) -> list[str]:
    return [chunk async for chunk in stream]


@pytest.mark.asyncio
async def test_streams_and_persists_user_and_assistant_messages() -> None:
    use_case, repository, agent, logger, metrics = build_use_case()
    store_id = uuid4()
    user_id = uuid4()
    now = datetime.now(UTC)

    chunks = await collect(
        use_case.execute(
            store_id=store_id,
            user_id=user_id,
            conversation_id=None,
            message="Where is my order?",
            system_prompt="You are a helpful store assistant.",
        )
    )

    assert chunks == ["Hello", " there!"]
    assert len(repository.messages) == 2

    user_message, assistant_message = repository.messages

    assert user_message.role is MessageRole.USER
    assert user_message.content.value == "Where is my order?"
    assert user_message.store_id == store_id
    assert user_message.user_id == user_id

    assert assistant_message.role is MessageRole.ASSISTANT
    assert assistant_message.content.value == "Hello there!"
    assert assistant_message.store_id == store_id
    assert assistant_message.user_id == user_id

    assert user_message.conversation_id == assistant_message.conversation_id
    assert user_message.conversation_id is not None
    assert user_message.id != assistant_message.id
    assert user_message.created_at.tzinfo is not None
    assert assistant_message.created_at.tzinfo is not None
    assert user_message.created_at >= now
    assert len(agent.calls) == 1
    assert agent.calls[0]["store_id"] == store_id
    assert agent.calls[0]["conversation_id"] == user_message.conversation_id
    assert agent.calls[0]["current_message_id"] == user_message.id
    assert agent.calls[0]["message"] == "Where is my order?"
    assert metrics.counters["chat.messages.user.persisted"] == 1
    assert metrics.counters["chat.messages.assistant.persisted"] == 1
    assert metrics.counters["chat.streams.completed"] == 1
    assert logger.events


@pytest.mark.asyncio
async def test_reuses_supplied_conversation_id() -> None:
    use_case, repository, _, _, _ = build_use_case()
    conversation_id = uuid4()

    await collect(
        use_case.execute(
            store_id=uuid4(),
            user_id=None,
            conversation_id=conversation_id,
            message="Hello",
            system_prompt="Be helpful.",
        )
    )

    assert len(repository.messages) == 2
    assert all(
        message.conversation_id == conversation_id
        for message in repository.messages
    )
    assert all(message.user_id is None for message in repository.messages)


@pytest.mark.asyncio
async def test_persists_user_message_before_agent_starts() -> None:
    repository = StubMessageRepository()

    class InspectingAgent(StubAgent):
        async def run(
            self,
            store_id: UUID,
            conversation_id: UUID,
            current_message_id: UUID,
            message: str,
            system_prompt: str,
        ) -> AsyncIterator[str]:
            assert len(repository.messages) == 1
            assert repository.messages[0].id == current_message_id
            async for chunk in super().run(
                store_id,
                conversation_id,
                current_message_id,
                message,
                system_prompt,
            ):
                yield chunk

    agent = InspectingAgent(["OK"])
    use_case = StreamMessage(
        message_repository=repository,
        agent=agent,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    assert await collect(
        use_case.execute(
            store_id=uuid4(),
            user_id=None,
            conversation_id=None,
            message="Hello",
            system_prompt="Be helpful.",
        )
    ) == ["OK"]

    assert len(repository.messages) == 2


@pytest.mark.asyncio
async def test_does_not_persist_assistant_message_when_agent_fails() -> None:
    use_case, repository, agent, logger, metrics = build_use_case(chunks=["Partial"])
    agent.error = RuntimeError("agent failed")

    with pytest.raises(RuntimeError, match="agent failed"):
        await collect(
            use_case.execute(
                store_id=uuid4(),
                user_id=None,
                conversation_id=None,
                message="Hello",
                system_prompt="Be helpful.",
            )
        )

    assert len(repository.messages) == 1
    assert repository.messages[0].role is MessageRole.USER
    assert metrics.counters["chat.streams.failed"] == 1
    assert any(level == "error" for level, _, _ in logger.events)


@pytest.mark.asyncio
async def test_does_not_start_agent_when_user_message_cannot_be_saved() -> None:
    use_case, repository, agent, _, metrics = build_use_case()
    repository.fail_on_save = True

    with pytest.raises(RuntimeError, match="message persistence failed"):
        await collect(
            use_case.execute(
                store_id=uuid4(),
                user_id=None,
                conversation_id=None,
                message="Hello",
                system_prompt="Be helpful.",
            )
        )

    assert agent.calls == []
    assert repository.messages == []
    assert metrics.counters.get("chat.streams.failed", 0) == 1


@pytest.mark.asyncio
async def test_rejects_empty_agent_response_without_saving_assistant_message() -> None:
    use_case, repository, _, _, metrics = build_use_case(chunks=[])

    with pytest.raises(ValueError):
        await collect(
            use_case.execute(
                store_id=uuid4(),
                user_id=None,
                conversation_id=None,
                message="Hello",
                system_prompt="Be helpful.",
            )
        )

    assert len(repository.messages) == 1
    assert repository.messages[0].role is MessageRole.USER
    assert metrics.counters["chat.streams.failed"] == 1
