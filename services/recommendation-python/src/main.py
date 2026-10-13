"""Recommendation Service process entry point."""

from __future__ import annotations

import asyncio
import logging

import uvicorn

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
from infrastructure.embedding.grpc_client import EmbeddingGRPCClient
from infrastructure.observability.logger import create_production_logger
from infrastructure.observability.metrics import Metrics
from infrastructure.reranking.rrf_reranker import RRFReranker
from presentation.http.app import create_app


def create_application() -> tuple[object, str, EmbeddingGRPCClient]:
    """Construct the HTTP application and its infrastructure dependencies."""
    settings = load_settings()
    logger = create_production_logger()
    metrics = Metrics()

    embedding_client = EmbeddingGRPCClient(
        address=settings.embedding_service.address,
        service_name=settings.embedding_service.service_name,
        service_secret=settings.embedding_service.service_secret,
        timeout_seconds=settings.embedding_service.timeout_seconds,
    )

    find_similar_stores = FindSimilarStores(
        similarity_retriever=embedding_client,
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
        similarity_retriever=embedding_client,
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

    http_app = create_app(
        service_name=settings.service.name,
        service_secret=settings.service.secret,
        recommend_stores=recommend_stores,
        recommend_products=recommend_products,
        logger=logger,
        metrics=metrics,
    )
    return http_app, settings.http.address, embedding_client


async def run_application(
    http_app: object,
    http_address: str,
    embedding_client: EmbeddingGRPCClient,
) -> None:
    """Run HTTP and close the outbound gRPC channel on shutdown."""
    host, separator, port_text = http_address.rpartition(":")
    if not separator or not host or not port_text:
        raise ValueError(
            "RECOMMENDATION_HTTP_ADDR must use host:port format"
        )

    server = uvicorn.Server(
        uvicorn.Config(
            http_app,
            host=host,
            port=int(port_text),
            log_level="info",
        )
    )
    try:
        await server.serve()
    finally:
        embedding_client.close()


def main() -> None:
    """Run the Recommendation service."""
    logging.basicConfig(level=logging.INFO)
    http_app, http_address, embedding_client = create_application()
    asyncio.run(
        run_application(
            http_app=http_app,
            http_address=http_address,
            embedding_client=embedding_client,
        )
    )


if __name__ == "__main__":
    main()
