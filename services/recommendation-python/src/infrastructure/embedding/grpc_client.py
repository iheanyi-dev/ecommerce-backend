"""Authenticated synchronous gRPC client for the Embedding service."""

from __future__ import annotations

from uuid import UUID

import grpc

from embedding.v1 import embedding_pb2, embedding_pb2_grpc


class EmbeddingGRPCClient:
    """Retrieve semantic candidates using Chat's authorized identity.

    Embedding currently authorizes one configured caller. Recommendation
    deliberately uses the same service name and shared secret as Chat.
    """

    def __init__(
        self,
        address: str,
        service_name: str,
        service_secret: str,
        timeout_seconds: float = 10.0,
    ) -> None:
        if not address.strip():
            raise ValueError("Embedding gRPC address must not be empty")
        if not service_name.strip():
            raise ValueError("Embedding caller service name must not be empty")
        if not service_secret:
            raise ValueError("Embedding caller service secret must not be empty")
        if timeout_seconds <= 0:
            raise ValueError("Embedding gRPC timeout must be positive")

        self._channel = grpc.insecure_channel(address.strip())
        self._stub = embedding_pb2_grpc.EmbeddingServiceStub(self._channel)
        self._metadata = (
            ("x-service-name", service_name.strip()),
            ("x-service-secret", service_secret),
        )
        self._timeout_seconds = timeout_seconds

    def find_similar_products(
        self,
        query: str,
        top_k: int,
        offset: int,
    ) -> list[tuple[UUID, float]]:
        """Request a page of semantically similar products."""
        response = self._stub.RetrieveSimilarProducts(
            embedding_pb2.RetrieveSimilarProductsRequest(
                query=query,
                top_k=top_k,
                offset=offset,
            ),
            metadata=self._metadata,
            timeout=self._timeout_seconds,
        )
        return [
            (UUID(item.product_id), float(item.score))
            for item in response.products
        ]

    def find_similar_stores(
        self,
        query: str,
        top_k: int,
        offset: int,
    ) -> list[tuple[UUID, float]]:
        """Request a page of semantically similar stores."""
        response = self._stub.RetrieveSimilarStores(
            embedding_pb2.RetrieveSimilarStoresRequest(
                query=query,
                top_k=top_k,
                offset=offset,
            ),
            metadata=self._metadata,
            timeout=self._timeout_seconds,
        )
        return [
            (UUID(item.store_id), float(item.score))
            for item in response.stores
        ]

    def close(self) -> None:
        """Release the underlying gRPC channel."""
        self._channel.close()
