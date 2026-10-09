"""Recommendation Service process entry point."""

from __future__ import annotations

import asyncio
import logging

import uvicorn

from application.use_cases.create_product_embedding import CreateProductEmbedding
from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_product_embedding import DeleteProductEmbedding
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding
from application.use_cases.find_associated_products import FindAssociatedProducts
from application.use_cases.find_associated_stores import FindAssociatedStores
from application.use_cases.find_similar_products import FindSimilarProducts
from application.use_cases.find_similar_stores import FindSimilarStores
from application.use_cases.recommend_products import RecommendProducts
from application.use_cases.recommend_stores import RecommendStores
from infrastructure.behavioral_models.empty_product_behavioral_model import (
    EmptyProductBehavioralAssociationModel,
)
from infrastructure.behavioral_models.empty_store_behavioral_model import (
    EmptyStoreBehavioralAssociationModel,
)
from infrastructure.config.settings import load_settings
from infrastructure.embeddings.gte_embedding_generator import GTEEmbeddingGenerator
from infrastructure.embeddings.model_loader import GTEModelLoader
from infrastructure.embeddings.model_manifest import GTE_MULTILINGUAL_BASE
from infrastructure.messaging.kafka.embedding_consumer import KafkaEmbeddingConsumer
from infrastructure.observability.logger import create_production_logger
from infrastructure.observability.metrics import Metrics
from infrastructure.persistence.qdrant.client import create_qdrant_client
from infrastructure.persistence.qdrant.collections import (
    PRODUCT_COLLECTION,
    STORE_COLLECTION,
    ensure_collection,
)
from infrastructure.persistence.qdrant.product_embedding_repository import (
    QdrantProductEmbeddingRepository,
)
from infrastructure.persistence.qdrant.store_embedding_repository import (
    QdrantStoreEmbeddingRepository,
)
from infrastructure.reranking.rrf_reranker import RRFReranker
from presentation.http.app import create_app


def create_application() -> tuple[
    object,
    str,
    KafkaEmbeddingConsumer,
]:
    """Build the complete Recommendation Service application."""

    settings = load_settings()

    logger = create_production_logger()
    metrics = Metrics()

    loaded_model = GTEModelLoader(
        manifest=GTE_MULTILINGUAL_BASE,
        cache_dir=settings.embedding.cache_dir,
    ).load()

    embedding_generator = GTEEmbeddingGenerator(
        loaded_model=loaded_model,
        manifest=GTE_MULTILINGUAL_BASE,
    )

    qdrant_client = create_qdrant_client(
        url=settings.qdrant.url,
        api_key=settings.qdrant.api_key,
    )

    ensure_collection(
        client=qdrant_client,
        collection_name=STORE_COLLECTION,
        vector_dimension=GTE_MULTILINGUAL_BASE.embedding_dimension,
    )
    ensure_collection(
        client=qdrant_client,
        collection_name=PRODUCT_COLLECTION,
        vector_dimension=GTE_MULTILINGUAL_BASE.embedding_dimension,
    )

    store_embedding_repository = QdrantStoreEmbeddingRepository(
        qdrant_client,
    )
    product_embedding_repository = QdrantProductEmbeddingRepository(
        qdrant_client,
    )

    create_store_embedding = CreateStoreEmbedding(
        embedding_generator=embedding_generator,
        store_embedding_repository=store_embedding_repository,
        logger=logger,
        metrics=metrics,
    )

    delete_store_embedding = DeleteStoreEmbedding(
        store_embedding_repository=store_embedding_repository,
        logger=logger,
        metrics=metrics,
    )

    create_product_embedding = CreateProductEmbedding(
        embedding_generator=embedding_generator,
        product_embedding_repository=product_embedding_repository,
        logger=logger,
        metrics=metrics,
    )

    delete_product_embedding = DeleteProductEmbedding(
        product_embedding_repository=product_embedding_repository,
        logger=logger,
        metrics=metrics,
    )

    find_similar_stores = FindSimilarStores(
        embedding_generator=embedding_generator,
        store_embedding_repository=store_embedding_repository,
        logger=logger,
        metrics=metrics,
    )

    find_associated_stores = FindAssociatedStores(
        store_behavioral_association_model=(
            EmptyStoreBehavioralAssociationModel()
        ),
        logger=logger,
        metrics=metrics,
    )

    find_similar_products = FindSimilarProducts(
        embedding_generator=embedding_generator,
        product_embedding_repository=product_embedding_repository,
        logger=logger,
        metrics=metrics,
    )

    find_associated_products = FindAssociatedProducts(
        product_behavioral_association_model=(
            EmptyProductBehavioralAssociationModel()
        ),
        logger=logger,
        metrics=metrics,
    )

    reranker = RRFReranker()

    recommend_stores = RecommendStores(
        find_similar_stores=find_similar_stores,
        find_associated_stores=find_associated_stores,
        reranker=reranker,
        logger=logger,
        metrics=metrics,
    )

    recommend_products = RecommendProducts(
        find_similar_products=find_similar_products,
        find_associated_products=find_associated_products,
        reranker=reranker,
        logger=logger,
        metrics=metrics,
    )

    kafka_consumer = KafkaEmbeddingConsumer(
        bootstrap_servers=settings.kafka.brokers,
        group_id=settings.kafka.consumer_group,
        create_store_embedding=create_store_embedding,
        delete_store_embedding=delete_store_embedding,
    )

    http_app = create_app(
        service_name=settings.service.name,
        service_secret=settings.service.secret,
        recommend_stores=recommend_stores,
        recommend_products=recommend_products,
        logger=logger,
        metrics=metrics,
    )

    return (
        http_app,
        settings.http.address,
        kafka_consumer,
    )


async def run_application(
    http_app: object,
    http_address: str,
    kafka_consumer: KafkaEmbeddingConsumer,
) -> None:
    """Run the Recommendation HTTP server and Kafka consumer."""

    http_host, separator, http_port_text = http_address.rpartition(":")
    if not separator or not http_host or not http_port_text:
        raise ValueError(
            "RECOMMENDATION_HTTP_ADDR must use host:port format"
        )

    http_port = int(http_port_text)

    http_config = uvicorn.Config(
        http_app,
        host=http_host,
        port=http_port,
        log_level="info",
    )
    http_server = uvicorn.Server(http_config)

    await kafka_consumer.start()

    try:
        await http_server.serve()
    finally:
        await kafka_consumer.stop()


def main() -> None:
    """Start the Recommendation HTTP server and Kafka consumer."""

    logging.basicConfig(level=logging.INFO)

    http_app, http_address, kafka_consumer = create_application()

    asyncio.run(
        run_application(
            http_app=http_app,
            http_address=http_address,
            kafka_consumer=kafka_consumer,
        )
    )


if __name__ == "__main__":
    main()
