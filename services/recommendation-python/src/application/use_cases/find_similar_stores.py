"""Use case for finding stores similar to a query."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_similar_stores import (
    FindSimilarStoresInput,
    FindSimilarStoresOutput,
    StoreRecommendation,
)
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.similarity_retriever import SimilarityRetriever

PAGE_SIZE = 10


@dataclass(slots=True)
class FindSimilarStores:
    """Find semantic store candidates through Embedding."""

    similarity_retriever: SimilarityRetriever
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: FindSimilarStoresInput,
    ) -> FindSimilarStoresOutput:
        """Retrieve the requested page of similar stores."""
        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        recommendations = self.similarity_retriever.find_similar_stores(
            query=input.query,
            top_k=PAGE_SIZE,
            offset=(input.page - 1) * PAGE_SIZE,
        )
        output = FindSimilarStoresOutput(
            recommendations=tuple(
                StoreRecommendation(store_id=store_id, score=score)
                for store_id, score in recommendations
            )
        )

        self.metrics.increment("recommendation.store_similarity.queries")
        self.logger.info(
            "recommendation.store_similarity.found",
            page=input.page,
            result_count=len(output.recommendations),
        )
        return output
