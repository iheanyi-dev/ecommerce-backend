"""Entrypoint for the internal Chat gRPC service."""

import asyncio
import logging

import grpc

from chat.v1 import chat_pb2_grpc
from presentation.dependencies.chat import ChatDependencies
from presentation.grpc.chat_service import ChatService
from shared.config import get_settings


async def serve() -> None:
    settings = get_settings()
    logging.basicConfig(level=settings.CHAT_LOG_LEVEL.upper())

    prompt = settings.load_system_prompt()
    dependencies = ChatDependencies(settings)

    server = grpc.aio.server(
        maximum_concurrent_rpcs=None,
    )
    chat_pb2_grpc.add_ChatServiceServicer_to_server(
        ChatService(
            stream_message=dependencies.stream_message,
            settings=settings,
            system_prompt=prompt,
        ),
        server,
    )

    address = f"{settings.CHAT_SERVICE_HOST}:{settings.CHAT_SERVICE_PORT}"
    server.add_insecure_port(address)

    try:
        await server.start()
        logging.getLogger("chat").info("chat.grpc.started", extra={"address": address})
        await server.wait_for_termination()
    finally:
        await server.stop(grace=5)
        await dependencies.aclose()


if __name__ == "__main__":
    asyncio.run(serve())
