"""DTOs for finding Stores associated by user behavior."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from .find_similar_stores import StoreRecommendation


@dataclass(frozen=True, slots=True)
class FindAssociatedStoresInput:
    """Primitive input for Store behavioral association search."""

    store_id: UUID
    page: int


@dataclass(frozen=True, slots=True)
class FindAssociatedStoresOutput:
    """Stores returned for the requested behavioral recommendation page."""

    recommendations: tuple[StoreRecommendation, ...]
