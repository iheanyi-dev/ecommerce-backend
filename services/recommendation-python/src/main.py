"""Recommendation Service process entry point."""

from __future__ import annotations

import logging
from concurrent import futures

import grpc

from application.use_cases.create_product_embedding import CreateProductEmbedding
from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_product_embedding import DeleteProductEmbedding
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding
from infrastructure.config.settings import load_settings
from infrastructure.embeddings.gte_embedding_generator import GTEEmbeddingGenerator
from infrastructure.embeddings.model_loader import GTEModelLoader
from infrastructure.embeddings.model_manifest import GTE_MULTILINGUAL_BASE
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
from infrastructure.proto.recommendation.v1 import recommendation_pb2_grpc
from presentation.grpc.auth import ServiceAuthInterceptor
from presentation.grpc.recommendation_server import RecommendationGrpcServer


def create_server() -> grpc.Server:
    """Build the fully configured Recommendation gRPC server."""

    settings = load_settings()

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

    store_embedding_repository = QdrantStoreEmbeddingRepository(qdrant_client)
    product_embedding_repository = QdrantProductEmbeddingRepository(qdrant_client)

    create_store_embedding = CreateStoreEmbedding(
        embedding_generator=embedding_generator,
        store_embedding_repository=store_embedding_repository,
    )
    delete_store_embedding = DeleteStoreEmbedding(
        store_embedding_repository=store_embedding_repository,
    )
    create_product_embedding = CreateProductEmbedding(
        embedding_generator=embedding_generator,
        product_embedding_repository=product_embedding_repository,
    )
    delete_product_embedding = DeleteProductEmbedding(
        product_embedding_repository=product_embedding_repository,
    )

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        interceptors=(
            ServiceAuthInterceptor(
                service_name=settings.grpc.service_name,
                service_secret=settings.grpc.service_secret,
            ),
        ),
    )

    recommendation_pb2_grpc.add_RecommendationServiceServicer_to_server(
        RecommendationGrpcServer(
            create_store_embedding=create_store_embedding,
            delete_store_embedding=delete_store_embedding,
            create_product_embedding=create_product_embedding,
            delete_product_embedding=delete_product_embedding,
        ),
        server,
    )

    server.add_insecure_port(settings.grpc.address)

    return server


def main() -> None:
    """Start the Recommendation Service."""

    logging.basicConfig(level=logging.INFO)

    server = create_server()
    server.start()

    logging.info("Recommendation gRPC server started")

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=5)


if __name__ == "__main__":
    main()
