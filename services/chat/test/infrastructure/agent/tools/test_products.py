"""Tests for the product search Agent tool."""

import json
from uuid import uuid4

import pytest

from infrastructure.agent.models import ToolContext
from infrastructure.agent.ports.product_search import (
    ProductSearchFilters,
    ProductSearchResult,
)
from infrastructure.agent.tools.products import SearchStoreProductsTool


class FakeProductSearch:
    def __init__(self, products):
        self.products = products
        self.store_id = None
        self.query = None
        self.filters = None

    async def search_products(self, *, store_id, query, filters):
        self.store_id = store_id
        self.query = query
        self.filters = filters
        return self.products


@pytest.fixture
def context():
    return ToolContext(
        store_id=uuid4(),
        conversation_id=uuid4(),
        current_message_id=uuid4(),
    )


@pytest.mark.asyncio
async def test_search_returns_products_for_trusted_store(context):
    search = FakeProductSearch(
        [
            ProductSearchResult(
                product_id="product-1",
                name="Travel Backpack",
                description="Water-resistant 30L backpack.",
                price="79.99",
                currency="USD",
                product_url="https://example.test/products/product-1",
                available=True,
            )
        ]
    )
    tool = SearchStoreProductsTool(product_search=search)

    result = json.loads(
        await tool.execute(
            arguments={"query": " waterproof backpack "},
            context=context,
        )
    )

    assert search.store_id == context.store_id
    assert search.query == "waterproof backpack"
    assert search.filters == ProductSearchFilters()
    assert result["status"] == "ok"
    assert result["products"][0]["product_id"] == "product-1"
    assert result["products"][0]["price"] == "79.99"
    assert result["products"][0]["available"] is True


@pytest.mark.asyncio
async def test_search_passes_strict_maximum_price_and_currency(context):
    search = FakeProductSearch(products=[])
    tool = SearchStoreProductsTool(product_search=search)

    result = json.loads(
        await tool.execute(
            arguments={
                "query": "black sneakers",
                "filters": {
                    "max_price": "30.00",
                    "currency": "USD",
                    "max_price_inclusive": False,
                    "available_only": True,
                },
            },
            context=context,
        )
    )

    assert result == {"status": "no_matching_products", "products": []}
    assert search.filters == ProductSearchFilters(
        max_price="30.00",
        currency="USD",
        max_price_inclusive=False,
        available_only=True,
    )


@pytest.mark.asyncio
async def test_search_passes_price_range(context):
    search = FakeProductSearch(products=[])
    tool = SearchStoreProductsTool(product_search=search)

    await tool.execute(
        arguments={
            "query": "shoes",
            "filters": {
                "min_price": "10",
                "max_price": "30",
                "currency": "NGN",
                "min_price_inclusive": False,
                "max_price_inclusive": True,
            },
        },
        context=context,
    )

    assert search.filters == ProductSearchFilters(
        min_price="10",
        max_price="30",
        currency="NGN",
        min_price_inclusive=False,
        max_price_inclusive=True,
    )


@pytest.mark.asyncio
async def test_search_reports_no_matching_products(context):
    tool = SearchStoreProductsTool(product_search=FakeProductSearch(products=[]))

    result = json.loads(
        await tool.execute(
            arguments={"query": "nonexistent product"},
            context=context,
        )
    )

    assert result == {"status": "no_matching_products", "products": []}


@pytest.mark.asyncio
@pytest.mark.parametrize("query", ["", "  ", None, 123])
async def test_search_rejects_invalid_query(context, query):
    search = FakeProductSearch(products=[])
    tool = SearchStoreProductsTool(product_search=search)

    result = json.loads(
        await tool.execute(arguments={"query": query}, context=context)
    )

    assert result == {"status": "invalid_query", "products": []}
    assert search.store_id is None
    assert search.query is None


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "filters",
    [
        {"max_price": "30"},
        {"max_price": "-1", "currency": "USD"},
        {"max_price": "NaN", "currency": "USD"},
        {"min_price": "40", "max_price": "30", "currency": "USD"},
        {"max_price": "30", "currency": "usd"},
        {"available_only": "yes"},
        {"max_price": "30", "currency": "USD", "unexpected": True},
        {"max_price_inclusive": False},
    ],
)
async def test_search_rejects_invalid_filters_without_calling_service(context, filters):
    search = FakeProductSearch(products=[])
    tool = SearchStoreProductsTool(product_search=search)

    result = json.loads(
        await tool.execute(
            arguments={"query": "shoes", "filters": filters},
            context=context,
        )
    )

    assert result == {"status": "invalid_filters", "products": []}
    assert search.store_id is None


@pytest.mark.asyncio
async def test_search_propagates_service_failures(context):
    class FailingSearch:
        async def search_products(self, *, store_id, query, filters):
            raise RuntimeError("service unavailable")

    tool = SearchStoreProductsTool(product_search=FailingSearch())

    with pytest.raises(RuntimeError, match="service unavailable"):
        await tool.execute(
            arguments={"query": "backpack"},
            context=context,
        )
