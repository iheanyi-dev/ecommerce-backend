"""Tests for the product service gRPC adapter."""

from uuid import uuid4

import grpc
import pytest

from infrastructure.agent.ports.product_search import ProductSearchFilters
from infrastructure.products.grpc_client import (
    ProductSearchError,
    ProductServiceGRPCClient,
)
from product.v1 import product_pb2


class FakeUnaryCall:
    def __init__(self, response=None, error=None):
        self.response = response
        self.error = error
        self.request = None
        self.metadata = None
        self.timeout = None

    async def __call__(self, request, *, metadata, timeout):
        self.request = request
        self.metadata = metadata
        self.timeout = timeout
        if self.error is not None:
            raise self.error
        return self.response


class FakeStub:
    def __init__(self, call):
        self.SearchProducts = call


@pytest.mark.asyncio
async def test_search_products_sends_query_filters_and_service_auth():
    store_id = uuid4()
    call = FakeUnaryCall(
        response=product_pb2.SearchProductsResponse(
            products=[
                product_pb2.Product(
                    product_id="p-1",
                    name="Travel Backpack",
                    description="Water-resistant 30L backpack.",
                    price="79.99",
                    currency="USD",
                    product_url="https://example.test/products/p-1",
                    available=True,
                )
            ]
        )
    )
    client = ProductServiceGRPCClient(
        target="unused",
        service_name="chat",
        service_secret="test-secret",
        timeout=2.0,
        stub=FakeStub(call),
    )

    products = await client.search_products(
        store_id=store_id,
        query=" backpack ",
        filters=ProductSearchFilters(
            max_price="30.00",
            currency="USD",
            max_price_inclusive=False,
            available_only=True,
        ),
    )

    assert call.request.store_id == str(store_id)
    assert call.request.query == "backpack"
    assert call.request.filters.max_price == "30.00"
    assert call.request.filters.currency == "USD"
    assert call.request.filters.max_price_inclusive is False
    assert call.request.filters.available_only is True
    assert dict(call.metadata) == {
        "x-service-name": "chat",
        "x-service-secret": "test-secret",
    }
    assert call.timeout == 2.0
    assert len(products) == 1
    assert products[0].product_id == "p-1"
    assert products[0].price == "79.99"
    assert products[0].available is True


@pytest.mark.asyncio
async def test_search_products_preserves_absent_filter_presence():
    call = FakeUnaryCall(response=product_pb2.SearchProductsResponse())
    client = ProductServiceGRPCClient(
        target="unused",
        service_name="chat",
        service_secret="test-secret",
        stub=FakeStub(call),
    )

    await client.search_products(
        store_id=uuid4(),
        query="backpack",
        filters=ProductSearchFilters(),
    )

    assert not call.request.filters.HasField("min_price")
    assert not call.request.filters.HasField("max_price")
    assert not call.request.filters.HasField("available_only")


@pytest.mark.asyncio
async def test_search_products_wraps_grpc_errors():
    call = FakeUnaryCall(
        error=grpc.aio.AioRpcError(
            grpc.StatusCode.UNAVAILABLE,
            (),
            (),
            "unavailable",
        )
    )
    client = ProductServiceGRPCClient(
        target="unused",
        service_name="chat",
        service_secret="test-secret",
        stub=FakeStub(call),
    )

    with pytest.raises(ProductSearchError):
        await client.search_products(
            store_id=uuid4(),
            query="backpack",
            filters=ProductSearchFilters(),
        )


@pytest.mark.asyncio
async def test_search_products_rejects_blank_query():
    call = FakeUnaryCall(response=product_pb2.SearchProductsResponse())
    client = ProductServiceGRPCClient(
        target="unused",
        service_name="chat",
        service_secret="test-secret",
        stub=FakeStub(call),
    )

    with pytest.raises(ValueError, match="query"):
        await client.search_products(
            store_id=uuid4(),
            query="   ",
            filters=ProductSearchFilters(),
        )

    assert call.request is None
