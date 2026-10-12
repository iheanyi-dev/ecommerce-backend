"""Explicit construction of Chat application and infrastructure dependencies."""

import logging

from application.use_cases.stream_message import StreamMessage
from infrastructure.agent.agent import Agent
from infrastructure.agent.conversation_history import (
    RepositoryConversationHistory,
)
from infrastructure.agent.providers.gemini import GeminiLLMProvider
from infrastructure.agent.tools.knowledge import SearchStoreKnowledgeTool
from infrastructure.agent.tools.products import SearchStoreProductsTool
from infrastructure.knowledge.grpc_client import EmbeddingServiceGRPCClient
from infrastructure.persistence.postgres.database import session_factory
from infrastructure.persistence.postgres.message_repository import (
    SQLAlchemyMessageRepository,
)
from infrastructure.products.grpc_client import ProductServiceGRPCClient
from shared.config import Settings


class ServiceLogger:
    """Adapt Python logging to the application's structured logger port."""

    def __init__(self) -> None:
        self._logger = logging.getLogger("chat")

    def info(self, event: str, **fields: object) -> None:
        self._logger.info(event, extra={"fields": fields})

    def warning(self, event: str, **fields: object) -> None:
        self._logger.warning(event, extra={"fields": fields})

    def error(self, event: str, **fields: object) -> None:
        self._logger.error(event, extra={"fields": fields})


class ServiceMetrics:
    """Minimal metrics-port adapter until the shared metrics exporter is wired."""

    def increment(self, name: str, value: int = 1) -> None:
        logging.getLogger("chat.metrics").debug(
            "counter", extra={"metric": name, "value": value}
        )

    def observe(self, name: str, value: float) -> None:
        logging.getLogger("chat.metrics").debug(
            "observation", extra={"metric": name, "value": value}
        )


class ChatDependencies:
    """Own infrastructure clients and the StreamMessage use case."""

    def __init__(self, settings: Settings) -> None:
        if not settings.GEMINI_API_KEY.get_secret_value():
            raise RuntimeError("GEMINI_API_KEY must be configured")
        if not settings.CHAT_CALLER_SERVICE_SECRET.get_secret_value():
            raise RuntimeError("CHAT_CALLER_SERVICE_SECRET must be configured")
        if not settings.EMBEDDING_SERVICE_ADDR.strip():
            raise RuntimeError("EMBEDDING_SERVICE_ADDR must be configured")
        if not settings.PRODUCT_SERVICE_ADDR.strip():
            raise RuntimeError("PRODUCT_SERVICE_ADDR must be configured")

        self._logger = ServiceLogger()
        self._metrics = ServiceMetrics()

        self._provider = GeminiLLMProvider(
            model=settings.GEMINI_MODEL,
            api_key=settings.GEMINI_API_KEY.get_secret_value(),
        )

        self._knowledge_client = EmbeddingServiceGRPCClient(
            target=settings.EMBEDDING_SERVICE_ADDR,
            service_name=settings.EMBEDDING_SERVICE_NAME,
            service_secret=settings.EMBEDDING_SERVICE_SECRET.get_secret_value(),
        )
        self._product_client = ProductServiceGRPCClient(
            target=settings.PRODUCT_SERVICE_ADDR,
            service_name=settings.PRODUCT_SERVICE_NAME,
            service_secret=settings.PRODUCT_SERVICE_SECRET.get_secret_value(),
        )

        self._agent = Agent(
            provider=self._provider,
            tools=[
                SearchStoreKnowledgeTool(retrieval=self._knowledge_client),
                SearchStoreProductsTool(product_search=self._product_client),
            ],
            logger=self._logger,
            metrics=self._metrics,
            conversation_history=RepositoryConversationHistory(session_factory),
            history_limit=settings.CHAT_CONVERSATION_HISTORY_LIMIT,
            max_tool_rounds=settings.CHAT_AGENT_MAX_TOOL_ROUNDS,
        )

        self.stream_message = StreamMessage(
            message_repository=SQLAlchemyMessageRepository(session_factory),
            agent=self._agent,
            logger=self._logger,
            metrics=self._metrics,
        )

    async def aclose(self) -> None:
        """Release clients owned by the dependency container."""
        await self._knowledge_client.aclose()
        await self._product_client.aclose()
        await self._provider.aclose()
