"""gRPC adapter for the product service."""

from uuid import UUID

import grpc

from infrastructure.agent.ports.product_search import (
    ProductSearchFilters,
    ProductSearchResult,
)
from product.v1 import product_pb2, product_pb2_grpc


class ProductSearchError(RuntimeError):
    """Raised when the product service cannot complete a search."""


class ProductServiceGRPCClient:
    """Search products without depending on the product service internals."""

    def __init__(
        self,
        *,
        target: str,
        service_name: str,
        service_secret: str,
        timeout: float = 5.0,
        channel: grpc.aio.Channel | None = None,
        stub: product_pb2_grpc.ProductServiceStub | None = None,
    ) -> None:
        if not target.strip() and channel is None and stub is None:
            raise ValueError("Product service target must not be empty")
        if not service_name.strip():
            raise ValueError("Product service name must not be empty")
        if not service_secret:
            raise ValueError("Product service secret must not be empty")
        if timeout <= 0:
            raise ValueError("Product service timeout must be positive")

        self._service_name = service_name
        self._service_secret = service_secret
        self._timeout = timeout
        self._owned_channel = (
            grpc.aio.insecure_channel(target)
            if channel is None and stub is None
            else None
        )
        active_channel = channel or self._owned_channel
        self._stub = stub or product_pb2_grpc.ProductServiceStub(active_channel)

    async def search_products(
        self,
        *,
        store_id: UUID,
        query: str,
        filters: ProductSearchFilters,
    ) -> list[ProductSearchResult]:
        """Send the query and structured constraints over authenticated gRPC."""
        normalized_query = query.strip()
        if not normalized_query:
            raise ValueError("Product search query must not be empty")

        product_filters = product_pb2.ProductFilters(
            currency=filters.currency or "",
        )

        if filters.min_price is not None:
            product_filters.min_price = filters.min_price
            product_filters.min_price_inclusive = filters.min_price_inclusive

        if filters.max_price is not None:
            product_filters.max_price = filters.max_price
            product_filters.max_price_inclusive = filters.max_price_inclusive

        if filters.available_only is not None:
            product_filters.available_only = filters.available_only

        request = product_pb2.SearchProductsRequest(
            store_id=str(store_id),
            query=normalized_query,
            filters=product_filters,
        )
        metadata = (
            ("x-service-name", self._service_name),
            ("x-service-secret", self._service_secret),
        )

        try:
            response = await self._stub.SearchProducts(
                request,
                metadata=metadata,
                timeout=self._timeout,
            )
        except grpc.RpcError as exc:
            raise ProductSearchError("Product service request failed") from exc

        return [
            ProductSearchResult(
                product_id=product.product_id,
                name=product.name,
                description=product.description,
                price=product.price,
                currency=product.currency,
                product_url=product.product_url,
                available=product.available,
            )
            for product in response.products
        ]

    async def aclose(self) -> None:
        """Close the channel when this adapter owns it."""
        if self._owned_channel is not None:
            await self._owned_channel.close()
