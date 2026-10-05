"""Recommendation gRPC server construction."""

from __future__ import annotations

from concurrent import futures

import grpc

from application.use_cases.create_product_embedding import CreateProductEmbedding
from application.use_cases.create_store_embedding import CreateStoreEmbedding
from application.use_cases.delete_product_embedding import DeleteProductEmbedding
from application.use_cases.delete_store_embedding import DeleteStoreEmbedding
from infrastructure.proto.recommendation.v1 import recommendation_pb2_grpc

from .auth import ServiceAuthInterceptor
from .recommendation_server import RecommendationGrpcServer


def create_grpc_server(
    *,
    address: str,
    service_name: str,
    service_secret: str,
    create_store_embedding: CreateStoreEmbedding,
    delete_store_embedding: DeleteStoreEmbedding,
    create_product_embedding: CreateProductEmbedding,
    delete_product_embedding: DeleteProductEmbedding,
) -> grpc.Server:
    """Build the authenticated Recommendation gRPC server."""

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        interceptors=(
            ServiceAuthInterceptor(
                service_name=service_name,
                service_secret=service_secret,
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

    server.add_insecure_port(address)
    return server
