"""Port for reranking Store and Product recommendation candidates."""

from __future__ import annotations

from typing import Protocol

from ..dto.find_similar_products import ProductRecommendation
from ..dto.find_similar_stores import StoreRecommendation


class Reranker(Protocol):
    """Reranks recommendation candidates for each supported entity type.

    The implementation belongs to infrastructure because the actual
    reranking strategy may change independently of the application layer.

    The reranker owns combining candidates from the different
    recommendation sources and determining their final order.
    """

    def rerank_stores(
        self,
        similar_stores: list[StoreRecommendation],
        associated_stores: list[StoreRecommendation],
    ) -> list[StoreRecommendation]:
        """Combine and rerank Store recommendation candidates."""
        ...

    def rerank_products(
        self,
        similar_products: list[ProductRecommendation],
        associated_products: list[ProductRecommendation],
    ) -> list[ProductRecommendation]:
        """Combine and rerank Product recommendation candidates."""
        ...
