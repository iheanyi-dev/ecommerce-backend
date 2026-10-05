"""DTOs for finding Products associated by user behavior."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from .find_similar_products import ProductRecommendation


@dataclass(frozen=True, slots=True)
class FindAssociatedProductsInput:
    """Primitive input for Product behavioral association search."""

    product_id: UUID
    page: int


@dataclass(frozen=True, slots=True)
class FindAssociatedProductsOutput:
    """Products returned for the requested behavioral recommendation page."""

    recommendations: tuple[ProductRecommendation, ...]
