"""DTOs for the external Store recommendation use case."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from .find_similar_stores import StoreRecommendation


@dataclass(frozen=True, slots=True)
class RecommendStoresInput:
    """Input for Store recommendations.

    Either or both inputs may be supplied.

    - query requests semantic Store similarity candidates.
    - store_id requests behavioral Store association candidates.

    When an input is absent, its corresponding candidate collection is
    simply empty.
    """

    query: str | None = None
    store_id: UUID | None = None


@dataclass(frozen=True, slots=True)
class RecommendStoresOutput:
    """Final reranked Store recommendations."""

    recommendations: tuple[StoreRecommendation, ...]
