"""Internal streaming gRPC adapter for Store chat requests."""

import logging
from collections.abc import AsyncIterator
from uuid import UUID, uuid4

import grpc

from application.use_cases.stream_message import StreamMessage
from shared.config import Settings

from chat.v1 import chat_pb2, chat_pb2_grpc


logger = logging.getLogger(__name__)


class ChatService(chat_pb2_grpc.ChatServiceServicer):
    """Authenticate Store callers and stream Chat responses."""

    def __init__(
        self,
        *,
        stream_message: StreamMessage,
        settings: Settings,
        system_prompt: str,
    ) -> None:
        self._stream_message = stream_message
        self._settings = settings
        self._system_prompt = system_prompt

    async def StreamMessage(
        self,
        request: chat_pb2.StreamMessageRequest,
        context: grpc.aio.ServicerContext,
    ) -> AsyncIterator[chat_pb2.StreamMessageResponse]:
        """Validate the internal request and stream generated text."""
        metadata = {
            item.key.lower(): item.value
            for item in context.invocation_metadata()
        }

        expected_name = self._settings.CHAT_CALLER_SERVICE_NAME
        expected_secret = (
            self._settings.CHAT_CALLER_SERVICE_SECRET.get_secret_value()
        )

        if (
            not expected_secret
            or metadata.get("x-service-name") != expected_name
            or metadata.get("x-service-secret") != expected_secret
        ):
            await context.abort(
                grpc.StatusCode.UNAUTHENTICATED,
                "Invalid service credentials",
            )
            return

        try:
            store_id = UUID(request.store_id)
            user_id = UUID(request.user_id) if request.user_id else None
            conversation_id = (
                UUID(request.conversation_id)
                if request.conversation_id
                else None
            )
        except ValueError:
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT,
                "Request contains an invalid UUID",
            )
            return

        if not request.message.strip():
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT,
                "Message must not be empty",
            )
            return

        # Resolve the ID here so the Store receives it even for a new chat.
        resolved_conversation_id = conversation_id or uuid4()

        try:
            async for chunk in self._stream_message.execute(
                store_id=store_id,
                user_id=user_id,
                conversation_id=resolved_conversation_id,
                message=request.message,
                system_prompt=self._system_prompt,
            ):
                yield chat_pb2.StreamMessageResponse(
                    type="content",
                    content=chunk,
                    conversation_id=str(resolved_conversation_id),
                )

            yield chat_pb2.StreamMessageResponse(
                type="completed",
                content="",
                conversation_id=str(resolved_conversation_id),
            )
        except Exception:
            # Do not leak database, provider, or internal exception details.
            logger.exception(
                "chat.grpc.stream_failed",
                extra={
                    "store_id": str(store_id),
                    "conversation_id": str(resolved_conversation_id),
                },
            )
            await context.abort(
                grpc.StatusCode.INTERNAL,
                "Chat response generation failed",
            )
