"""Port for searching products belonging to a store."""

from dataclasses import dataclass
from typing import Protocol
from uuid import UUID


@dataclass(frozen=True, slots=True)
class ProductSearchFilters:
    """Explicit, validated product-search constraints."""

    min_price: str | None = None
    max_price: str | None = None
    currency: str | None = None
    min_price_inclusive: bool = True
    max_price_inclusive: bool = True
    available_only: bool | None = None


@dataclass(frozen=True, slots=True)
class ProductSearchResult:
    """Customer-safe product information returned by search."""

    product_id: str
    name: str
    description: str
    price: str
    currency: str
    product_url: str
    available: bool


class ProductSearch(Protocol):
    """Search the current store's product catalogue."""

    async def search_products(
        self,
        *,
        store_id: UUID,
        query: str,
        filters: ProductSearchFilters,
    ) -> list[ProductSearchResult]:
        """Return products matching the query and explicit constraints."""
        ...
