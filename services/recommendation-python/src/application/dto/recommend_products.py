"""DTOs for the external Product recommendation use case."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from .find_similar_products import ProductRecommendation


@dataclass(frozen=True, slots=True)
class RecommendProductsInput:
    """Input for Product recommendations.

    Either or both inputs may be supplied.

    - query requests semantic Product similarity candidates.
    - product_id requests behavioral Product association candidates.

    When an input is absent, its corresponding candidate collection is
    simply empty.
    """

    query: str | None = None
    product_id: UUID | None = None


@dataclass(frozen=True, slots=True)
class RecommendProductsOutput:
    """Final reranked Product recommendations."""

    recommendations: tuple[ProductRecommendation, ...]
