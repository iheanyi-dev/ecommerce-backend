"""DTOs for finding Stores similar to a query."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class FindSimilarStoresInput:
    """Primitive input for Store similarity search."""

    query: str
    page: int


@dataclass(frozen=True, slots=True)
class StoreRecommendation:
    """A Store recommendation candidate and its similarity score."""

    store_id: UUID
    score: float


@dataclass(frozen=True, slots=True)
class FindSimilarStoresOutput:
    """Stores returned for the requested recommendation page."""

    recommendations: tuple[StoreRecommendation, ...]
