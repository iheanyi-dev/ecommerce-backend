"""External use case for Store recommendations."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_associated_stores import FindAssociatedStoresInput
from ..dto.find_similar_stores import (
    FindSimilarStoresInput,
)
from ..dto.recommend_stores import (
    RecommendStoresInput,
    RecommendStoresOutput,
)
from ..ports.reranker import Reranker
from .find_associated_stores import FindAssociatedStores
from .find_similar_stores import FindSimilarStores


@dataclass(slots=True)
class RecommendStores:
    """Orchestrate Store recommendation candidate generation and reranking.

    This is the external Store recommendation application use case.

    Candidate generation remains delegated to the dedicated internal
    use cases. The reranker owns combining the different candidate
    collections and determining their final order.

    No pagination or ranking strategy belongs here.
    """

    find_similar_stores: FindSimilarStores
    find_associated_stores: FindAssociatedStores
    reranker: Reranker

    def execute(
        self,
        input: RecommendStoresInput,
    ) -> RecommendStoresOutput:
        """Generate available candidates and return their final ranking."""

        similar_stores = []

        if input.query is not None:
            result = self.find_similar_stores.execute(
                FindSimilarStoresInput(
                    query=input.query,
                    page=1,
                )
            )
            similar_stores = list(result.recommendations)

        associated_stores = []

        if input.store_id is not None:
            result = self.find_associated_stores.execute(
                FindAssociatedStoresInput(
                    store_id=input.store_id,
                    page=1,
                )
            )
            associated_stores = list(result.recommendations)

        recommendations = self.reranker.rerank_stores(
            similar_stores=similar_stores,
            associated_stores=associated_stores,
        )

        return RecommendStoresOutput(
            recommendations=tuple(recommendations),
        )
