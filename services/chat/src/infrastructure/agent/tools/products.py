"""Agent tool for searching the current store's product catalogue."""

import json
import re
from decimal import Decimal, InvalidOperation
from typing import Any

from infrastructure.agent.models import ToolContext
from infrastructure.agent.ports.product_search import (
    ProductSearch,
    ProductSearchFilters,
)


_PRICE_PATTERN = re.compile(r"^(?:0|[1-9]\d*)(?:\.\d+)?$")
_CURRENCY_PATTERN = re.compile(r"^[A-Z]{3}$")
_FILTER_KEYS = {
    "min_price",
    "max_price",
    "currency",
    "min_price_inclusive",
    "max_price_inclusive",
    "available_only",
}


class SearchStoreProductsTool:
    """Search store products using explicit customer constraints."""

    name = "search_store_products"
    description = (
        "Search the current store's product catalogue. Extract price limits, "
        "currency, and availability preferences into the filters object. "
        "Use min_price and max_price as decimal strings. Set the corresponding "
        "inclusive flag to false for strict comparisons: for example, a price "
        "strictly below 30 USD uses max_price='30', currency='USD', and "
        "max_price_inclusive=false. Never invent product details or prices."
    )

    input_schema: dict[str, Any] = {
        "type": "object",
        "properties": {
            "query": {
                "type": "string",
                "description": "The product, features, or category requested.",
            },
            "filters": {
                "type": "object",
                "description": "Explicit price, currency, and availability constraints.",
                "properties": {
                    "min_price": {
                        "type": "string",
                        "pattern": r"^(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$",
                        "description": "Inclusive or exclusive lower price boundary.",
                    },
                    "max_price": {
                        "type": "string",
                        "pattern": r"^(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$",
                        "description": "Inclusive or exclusive upper price boundary.",
                    },
                    "currency": {
                        "type": "string",
                        "pattern": "^[A-Z]{3}$",
                        "description": "Three-letter currency code, required for price filters.",
                    },
                    "min_price_inclusive": {
                        "type": "boolean",
                        "description": "Whether the minimum price itself is included. Defaults to true.",
                    },
                    "max_price_inclusive": {
                        "type": "boolean",
                        "description": "Whether the maximum price itself is included. Defaults to true.",
                    },
                    "available_only": {
                        "type": "boolean",
                        "description": "When true, return only available products.",
                    },
                },
                "additionalProperties": False,
            },
        },
        "required": ["query"],
        "additionalProperties": False,
    }

    def __init__(self, *, product_search: ProductSearch) -> None:
        self._product_search = product_search

    async def execute(
        self,
        *,
        arguments: dict[str, Any],
        context: ToolContext,
    ) -> str:
        query = arguments.get("query")
        if not isinstance(query, str) or not query.strip():
            return self._result("invalid_query")

        try:
            filters = self._parse_filters(arguments.get("filters"))
        except (TypeError, ValueError):
            return self._result("invalid_filters")

        products = await self._product_search.search_products(
            store_id=context.store_id,
            query=query.strip(),
            filters=filters,
        )

        return json.dumps(
            {
                "status": "ok" if products else "no_matching_products",
                "products": [
                    {
                        "product_id": product.product_id,
                        "name": product.name,
                        "description": product.description,
                        "price": product.price,
                        "currency": product.currency,
                        "product_url": product.product_url,
                        "available": product.available,
                    }
                    for product in products
                ],
            },
            ensure_ascii=False,
        )

    @staticmethod
    def _result(status: str) -> str:
        return json.dumps(
            {"status": status, "products": []},
            ensure_ascii=False,
        )

    @staticmethod
    def _parse_filters(raw: Any) -> ProductSearchFilters:
        if raw is None:
            raw = {}
        if not isinstance(raw, dict) or raw.keys() - _FILTER_KEYS:
            raise ValueError("Invalid product filters")

        min_price = SearchStoreProductsTool._parse_price(raw.get("min_price"))
        max_price = SearchStoreProductsTool._parse_price(raw.get("max_price"))

        currency = raw.get("currency")
        if currency is not None:
            if not isinstance(currency, str) or not _CURRENCY_PATTERN.fullmatch(currency):
                raise ValueError("Currency must be a three-letter uppercase code")

        if (min_price is not None or max_price is not None) and currency is None:
            raise ValueError("Currency is required for price filters")

        if min_price is not None and max_price is not None:
            if Decimal(min_price) > Decimal(max_price):
                raise ValueError("Minimum price cannot exceed maximum price")

        min_inclusive = raw.get("min_price_inclusive", True)
        max_inclusive = raw.get("max_price_inclusive", True)
        available_only = raw.get("available_only")

        if not isinstance(min_inclusive, bool) or not isinstance(max_inclusive, bool):
            raise ValueError("Price inclusivity flags must be booleans")
        if available_only is not None and not isinstance(available_only, bool):
            raise ValueError("available_only must be a boolean")

        if min_price is None and "min_price_inclusive" in raw:
            raise ValueError("min_price_inclusive requires min_price")
        if max_price is None and "max_price_inclusive" in raw:
            raise ValueError("max_price_inclusive requires max_price")

        return ProductSearchFilters(
            min_price=min_price,
            max_price=max_price,
            currency=currency,
            min_price_inclusive=min_inclusive,
            max_price_inclusive=max_inclusive,
            available_only=available_only,
        )

    @staticmethod
    def _parse_price(value: Any) -> str | None:
        if value is None:
            return None
        if not isinstance(value, str) or not _PRICE_PATTERN.fullmatch(value):
            raise ValueError("Price must be a non-negative decimal string")
        try:
            parsed = Decimal(value)
        except InvalidOperation as exc:
            raise ValueError("Invalid price") from exc
        if not parsed.is_finite():
            raise ValueError("Price must be finite")
        return format(parsed, "f")
