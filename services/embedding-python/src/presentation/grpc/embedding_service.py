"""Authenticated gRPC adapter for tenant-scoped knowledge retrieval."""

from __future__ import annotations

import asyncio
import logging
from pathlib import PurePosixPath
from uuid import UUID

import grpc

from application.dto.retrieve_knowledge import RetrieveKnowledgeInput
from application.use_cases.retrieve_knowledge import RetrieveKnowledge
from embedding.v1 import embedding_pb2, embedding_pb2_grpc
from infrastructure.config.settings import Settings

logger = logging.getLogger(__name__)


class EmbeddingService(embedding_pb2_grpc.EmbeddingServiceServicer):
    """Expose knowledge retrieval to trusted internal services."""

    def __init__(
        self,
        *,
        retrieve_knowledge: RetrieveKnowledge,
        settings: Settings,
    ) -> None:
        self._retrieve_knowledge = retrieve_knowledge
        self._settings = settings

    async def RetrieveChunks(
        self,
        request: embedding_pb2.RetrieveChunksRequest,
        context: grpc.aio.ServicerContext,
    ) -> embedding_pb2.RetrieveChunksResponse:
        """Authenticate, validate, and return relevant knowledge chunks."""
        metadata = {
            item.key.lower(): item.value
            for item in context.invocation_metadata()
        }
        expected_name = self._settings.grpc.caller_service_name
        expected_secret = self._settings.grpc.caller_service_secret

        if (
            not expected_secret
            or metadata.get("x-service-name") != expected_name
            or metadata.get("x-service-secret") != expected_secret
        ):
            await context.abort(
                grpc.StatusCode.UNAUTHENTICATED,
                "Invalid service credentials",
            )
            raise AssertionError("context.abort must terminate the RPC")

        try:
            store_id = UUID(request.store_id)
        except (ValueError, AttributeError):
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT,
                "store_id must be a valid UUID",
            )
            raise AssertionError("context.abort must terminate the RPC")

        if not request.query.strip():
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT,
                "query must not be empty",
            )
            raise AssertionError("context.abort must terminate the RPC")

        try:
            # The current model and Qdrant adapters are synchronous. Keep their
            # inference and I/O work off the gRPC asyncio event loop.
            result = await asyncio.to_thread(
                self._retrieve_knowledge.execute,
                RetrieveKnowledgeInput(store_id=store_id, query=request.query),
            )
        except ValueError as error:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(error))
            raise AssertionError("context.abort must terminate the RPC")
        except Exception:
            logger.exception(
                "embedding.grpc.retrieval_failed",
                extra={"store_id": request.store_id},
            )
            await context.abort(
                grpc.StatusCode.INTERNAL,
                "Knowledge retrieval failed",
            )
            raise AssertionError("context.abort must terminate the RPC")

        return embedding_pb2.RetrieveChunksResponse(
            chunks=[
                embedding_pb2.KnowledgeChunk(
                    content=item.chunk.content,
                    source_filename=PurePosixPath(
                        item.chunk.storage_reference
                    ).name,
                    score=item.score,
                )
                for item in result.chunks
            ]
        )
