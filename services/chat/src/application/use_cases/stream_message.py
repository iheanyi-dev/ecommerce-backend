"""Stream a store assistant response and persist completed messages."""

from collections.abc import AsyncIterator
from datetime import UTC, datetime
from time import perf_counter
from uuid import UUID, uuid4

from application.ports.agent import Agent
from application.ports.logger import Logger
from application.ports.message_repository import MessageRepository
from application.ports.metrics import Metrics
from domain.entities.message import Message
from domain.value_objects.message_content import MessageContent
from domain.value_objects.message_role import MessageRole


class StreamMessage:
    """Coordinate message persistence and agent response streaming."""

    def __init__(
        self,
        *,
        message_repository: MessageRepository,
        agent: Agent,
        logger: Logger,
        metrics: Metrics,
    ) -> None:
        self._message_repository = message_repository
        self._agent = agent
        self._logger = logger
        self._metrics = metrics

    async def execute(
        self,
        *,
        store_id: UUID,
        user_id: UUID | None,
        conversation_id: UUID | None,
        message: str,
        system_prompt: str,
    ) -> AsyncIterator[str]:
        """Persist the customer message and stream the assistant response.

        The customer message remains persisted if generation fails. The
        assistant message is persisted only after successful, non-empty
        response generation and successful exhaustion of the response stream.
        """
        started_at = perf_counter()
        resolved_conversation_id = conversation_id or uuid4()
        user_message_id = uuid4()
        now = datetime.now(UTC)

        user_message = Message(
            id=user_message_id,
            conversation_id=resolved_conversation_id,
            store_id=store_id,
            user_id=user_id,
            role=MessageRole.USER,
            content=MessageContent(message),
            created_at=now,
            updated_at=now,
        )

        self._logger.info(
            "chat.stream.started",
            store_id=str(store_id),
            conversation_id=str(resolved_conversation_id),
            anonymous=user_id is None,
        )

        try:
            await self._message_repository.save(user_message)
            self._metrics.increment("chat.messages.user.persisted")

            chunks: list[str] = []

            async for chunk in self._agent.run(
                store_id=store_id,
                conversation_id=resolved_conversation_id,
                current_message_id=user_message_id,
                message=message,
                system_prompt=system_prompt,
            ):
                if not isinstance(chunk, str):
                    raise TypeError("agent response chunks must be strings")
                if chunk:
                    chunks.append(chunk)
                    yield chunk

            response = "".join(chunks)
            if not response.strip():
                raise ValueError("agent returned an empty response")

            completed_at = datetime.now(UTC)
            assistant_message = Message(
                id=uuid4(),
                conversation_id=resolved_conversation_id,
                store_id=store_id,
                user_id=user_id,
                role=MessageRole.ASSISTANT,
                content=MessageContent(response),
                created_at=completed_at,
                updated_at=completed_at,
            )

            await self._message_repository.save(assistant_message)
            self._metrics.increment("chat.messages.assistant.persisted")
            self._metrics.increment("chat.streams.completed")
            self._metrics.observe(
                "chat.stream.duration_seconds",
                perf_counter() - started_at,
            )
            self._logger.info(
                "chat.stream.completed",
                store_id=str(store_id),
                conversation_id=str(resolved_conversation_id),
                response_characters=len(response),
            )
        except Exception as exc:
            self._metrics.increment("chat.streams.failed")
            self._metrics.observe(
                "chat.stream.duration_seconds",
                perf_counter() - started_at,
            )
            self._logger.error(
                "chat.stream.failed",
                store_id=str(store_id),
                conversation_id=str(resolved_conversation_id),
                error_type=type(exc).__name__,
            )
            raise
