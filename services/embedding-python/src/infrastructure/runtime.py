"""Composition root and coordinated lifecycle for the Embedding service."""

from __future__ import annotations

import asyncio
import logging
import os
from dataclasses import dataclass
from typing import Any

import grpc
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from qdrant_client import QdrantClient

from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_knowledge import DeleteKnowledge
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding
from application.use_cases.ingest_knowledge import IngestKnowledge
from application.use_cases.retrieve_knowledge import RetrieveKnowledge
from application.use_cases.create_product_embedding import CreateProductEmbedding
from application.use_cases.delete_product_embedding import DeleteProductEmbedding
from application.use_cases.retrieve_product_embeddings import RetrieveProductEmbeddings
from application.use_cases.retrieve_store_embeddings import RetrieveStoreEmbeddings
from infrastructure.config.settings import Settings, load_settings
from infrastructure.embeddings.gte_embedding_model import GTEEmbeddingModel
from infrastructure.messaging.event_handlers import (
    EVENT_TOPICS,
    EmbeddingEventHandlers,
)
from infrastructure.messaging.kafka_consumer import KafkaEventConsumer
from infrastructure.observability.logger import create_production_logger
from infrastructure.observability.metrics import Metrics
from infrastructure.persistence.qdrant.collections import (
    ensure_knowledge_chunk_collection,
    ensure_product_embedding_collection,
    ensure_store_embedding_collection,
)
from infrastructure.persistence.qdrant.knowledge_chunk_repository import (
    QdrantKnowledgeChunkRepository,
)
from infrastructure.persistence.qdrant.product_embedding_repository import (
    QdrantProductEmbeddingRepository,
)
from infrastructure.persistence.qdrant.store_embedding_repository import (
    QdrantStoreEmbeddingRepository,
)
from infrastructure.storage_document_content_reader import (
    StorageDocumentContentReader,
)
from infrastructure.text_processing.text_chunker import CharacterTextChunker
from presentation.server import serve

logger = logging.getLogger("embedding.runtime")


@dataclass(slots=True)
class Runtime:
    """Own the service dependencies and closeable infrastructure resources."""

    settings: Settings
    retrieve_knowledge: RetrieveKnowledge
    retrieve_product_embeddings: RetrieveProductEmbeddings
    retrieve_store_embeddings: RetrieveStoreEmbeddings
    embedding_model: GTEEmbeddingModel
    consumer: KafkaEventConsumer
    storage_reader: StorageDocumentContentReader
    qdrant: QdrantClient
    app_logger: Any
    tracer_provider: TracerProvider

    def close(self) -> None:
        """Release synchronous resources after both workers have stopped."""
        try:
            self.storage_reader.close()
        finally:
            try:
                self.qdrant.close()
            finally:
                self.tracer_provider.shutdown()


def _configure_tracing() -> TracerProvider:
    """Install an SDK provider and enable OTLP export when configured."""
    provider = TracerProvider(
        resource=Resource.create({"service.name": "embedding"})
    )
    endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "").strip()
    if endpoint:
        provider.add_span_processor(
            BatchSpanProcessor(OTLPSpanExporter(endpoint=endpoint))
        )

    # Do not replace an already configured provider, which can happen in tests.
    try:
        trace.set_tracer_provider(provider)
    except Exception:
        provider.shutdown()
        current = trace.get_tracer_provider()
        if isinstance(current, TracerProvider):
            return current
        raise

    return provider


def build_runtime(settings: Settings | None = None) -> Runtime:
    """Construct all adapters, repositories, use cases, and event handlers."""
    settings = settings or load_settings()
    app_logger = create_production_logger()
    metrics = Metrics()
    embedding_model = GTEEmbeddingModel(
        cache_dir=settings.embedding.cache_dir,
        local_files_only=False,
    )

    qdrant = QdrantClient(
        url=settings.qdrant.url,
        api_key=settings.qdrant.api_key,
    )
    storage_reader: StorageDocumentContentReader | None = None

    try:
        # Collection dimensions must match the configured embedding model.
        from infrastructure.embeddings.model_manifest import (
            GTE_MULTILINGUAL_BASE,
        )

        dimension = GTE_MULTILINGUAL_BASE.embedding_dimension
        ensure_store_embedding_collection(qdrant, dimension)
        ensure_product_embedding_collection(qdrant, dimension)
        ensure_knowledge_chunk_collection(qdrant, dimension)

        store_repository = QdrantStoreEmbeddingRepository(qdrant)
        product_repository = QdrantProductEmbeddingRepository(qdrant)
        knowledge_repository = QdrantKnowledgeChunkRepository(qdrant)

        storage_reader = StorageDocumentContentReader.connect(settings.storage)

        create_store = CreateStoreEmbedding(
            embedding_model=embedding_model,
            store_embedding_repository=store_repository,
            logger=app_logger,
            metrics=metrics,
        )
        delete_store = DeleteStoreEmbedding(
            store_embedding_repository=store_repository,
            product_embedding_repository=product_repository,
            logger=app_logger,
            metrics=metrics,
        )
        ingest_knowledge = IngestKnowledge(
            document_content_reader=storage_reader,
            text_chunker=CharacterTextChunker(),
            embedding_model=embedding_model,
            knowledge_chunk_repository=knowledge_repository,
            logger=app_logger,
            metrics=metrics,
        )
        delete_knowledge = DeleteKnowledge(
            knowledge_chunk_repository=knowledge_repository,
            logger=app_logger,
            metrics=metrics,
        )
        retrieve_knowledge = RetrieveKnowledge(
            knowledge_chunk_repository=knowledge_repository,
            embedding_model=embedding_model,
            logger=app_logger,
            metrics=metrics,
        )
        create_product = CreateProductEmbedding(
            embedding_model=embedding_model,
            product_embedding_repository=product_repository,
            logger=app_logger,
            metrics=metrics,
        )
        delete_product = DeleteProductEmbedding(
            product_embedding_repository=product_repository,
            logger=app_logger,
            metrics=metrics,
        )

        retrieve_product_embeddings = RetrieveProductEmbeddings(
            embedding_model=embedding_model,
            product_embedding_repository=product_repository,
            logger=app_logger,
            metrics=metrics,
        )
        retrieve_store_embeddings = RetrieveStoreEmbeddings(
            embedding_model=embedding_model,
            store_embedding_repository=store_repository,
            logger=app_logger,
            metrics=metrics,
        )

        handlers = EmbeddingEventHandlers(
            create_store_embedding=create_store,
            delete_store_embedding=delete_store,
            ingest_knowledge=ingest_knowledge,
            delete_knowledge=delete_knowledge,
            create_product_embedding=create_product,
            delete_product_embedding=delete_product,
        )
        consumer = KafkaEventConsumer(
            brokers=settings.kafka.bootstrap_servers,
            group_id=settings.kafka.group_id,
            topics=EVENT_TOPICS,
            handler=handlers.handle,
            logger=app_logger,
        )

        return Runtime(
            settings=settings,
            embedding_model=embedding_model,
            retrieve_knowledge=retrieve_knowledge,
            retrieve_product_embeddings=retrieve_product_embeddings,
            retrieve_store_embeddings=retrieve_store_embeddings,
            consumer=consumer,
            storage_reader=storage_reader,
            qdrant=qdrant,
            app_logger=app_logger,
            tracer_provider=_configure_tracing(),
        )
    except Exception:
        if storage_reader is not None:
            storage_reader.close()
        qdrant.close()
        raise


async def run_runtime(runtime: Runtime) -> None:
    """Warm the model, then run gRPC and Kafka with coordinated lifecycle."""
    logger.info("embedding.model.warmup.started")
    await asyncio.to_thread(runtime.embedding_model.load)
    logger.info("embedding.model.warmup.completed")

    workers = {
        asyncio.create_task(
            serve(
                retrieve_knowledge=runtime.retrieve_knowledge,
                retrieve_product_embeddings=runtime.retrieve_product_embeddings,
                retrieve_store_embeddings=runtime.retrieve_store_embeddings,
                settings=runtime.settings,
            ),
            name="embedding-grpc",
        ),
        asyncio.create_task(runtime.consumer.run(), name="embedding-kafka"),
    }

    try:
        done, pending = await asyncio.wait(
            workers,
            return_when=asyncio.FIRST_COMPLETED,
        )

        # Surface worker failures after cancelling the sibling worker.
        for task in done:
            task.result()
    finally:
        for task in workers:
            if not task.done():
                task.cancel()
        await asyncio.gather(*workers, return_exceptions=True)
