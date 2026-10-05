"""DTOs for finding Products similar to a query."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class FindSimilarProductsInput:
    """Primitive input for Product similarity search."""

    query: str
    page: int


@dataclass(frozen=True, slots=True)
class ProductRecommendation:
    """A Product recommendation candidate and its similarity score."""

    product_id: UUID
    score: float


@dataclass(frozen=True, slots=True)
class FindSimilarProductsOutput:
    """Products returned for the requested recommendation page."""

    recommendations: tuple[ProductRecommendation, ...]
