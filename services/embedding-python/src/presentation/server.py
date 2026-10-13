"""Lifecycle helpers for the Embedding gRPC server."""

from __future__ import annotations

import asyncio
import logging

import grpc

from application.use_cases.retrieve_knowledge import RetrieveKnowledge
from application.use_cases.retrieve_product_embeddings import RetrieveProductEmbeddings
from application.use_cases.retrieve_store_embeddings import RetrieveStoreEmbeddings
from embedding.v1 import embedding_pb2_grpc
from infrastructure.config.settings import Settings
from presentation.grpc.embedding_service import EmbeddingService
from presentation.grpc.tracing_interceptor import TracingServerInterceptor

logger = logging.getLogger(__name__)


async def serve(
    *,
    retrieve_knowledge: RetrieveKnowledge,
    retrieve_product_embeddings: RetrieveProductEmbeddings,
    retrieve_store_embeddings: RetrieveStoreEmbeddings,
    settings: Settings,
) -> None:
    """Run the internal Embedding gRPC server until shutdown."""
    server = grpc.aio.server(
        interceptors=[TracingServerInterceptor("embedding")],
    )
    embedding_pb2_grpc.add_EmbeddingServiceServicer_to_server(
        EmbeddingService(
            retrieve_knowledge=retrieve_knowledge,
            retrieve_product_embeddings=retrieve_product_embeddings,
            retrieve_store_embeddings=retrieve_store_embeddings,
            settings=settings,
        ),
        server,
    )

    address = settings.grpc.address
    if server.add_insecure_port(address) == 0:
        raise RuntimeError(f"Could not bind Embedding gRPC server to {address}")

    try:
        await server.start()
        logger.info("embedding.grpc.started", extra={"address": address})
        await server.wait_for_termination()
    finally:
        await server.stop(grace=5)


def run(
    *,
    retrieve_knowledge: RetrieveKnowledge,
    retrieve_product_embeddings: RetrieveProductEmbeddings,
    retrieve_store_embeddings: RetrieveStoreEmbeddings,
    settings: Settings,
) -> None:
    """Synchronous entrypoint for an already-constructed application."""
    asyncio.run(
        serve(
            retrieve_knowledge=retrieve_knowledge,
            retrieve_product_embeddings=retrieve_product_embeddings,
            retrieve_store_embeddings=retrieve_store_embeddings,
            settings=settings,
        )
    )
