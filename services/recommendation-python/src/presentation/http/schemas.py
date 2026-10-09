"""HTTP schemas for the Recommendation Service."""

from __future__ import annotations

from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field


class RecommendationItem(BaseModel):
    """A single recommendation returned to an HTTP caller."""

    model_config = ConfigDict(frozen=True)

    entity_id: UUID
    score: float


class RecommendationResponse(BaseModel):
    """Final recommendation response."""

    model_config = ConfigDict(frozen=True)

    recommendations: tuple[RecommendationItem, ...]


class StoreRecommendationQuery(BaseModel):
    """Store recommendation request parameters."""

    query: str | None = Field(default=None, min_length=1)
    store_id: UUID | None = None


class ProductRecommendationQuery(BaseModel):
    """Product recommendation request parameters."""

    query: str | None = Field(default=None, min_length=1)
    product_id: UUID | None = None
