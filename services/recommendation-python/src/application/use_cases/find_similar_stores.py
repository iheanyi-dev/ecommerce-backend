"""Use case for finding Stores similar to a query."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_similar_stores import (
    FindSimilarStoresInput,
    FindSimilarStoresOutput,
    StoreRecommendation,
)
from ..ports.embedding_generator import EmbeddingGenerator
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.store_embedding_repository import StoreEmbeddingRepository

PAGE_SIZE = 10


@dataclass(slots=True)
class FindSimilarStores:
    """Find Store recommendations for a text query."""

    embedding_generator: EmbeddingGenerator
    store_embedding_repository: StoreEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: FindSimilarStoresInput,
    ) -> FindSimilarStoresOutput:
        """Generate the query embedding and return the requested page."""

        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        query_embedding = (
            self.embedding_generator.generate_store_query_embedding(
                input.query,
            )
        )

        candidates = self.store_embedding_repository.find_similar(
            embedding=query_embedding,
            limit=input.page * PAGE_SIZE,
        )

        offset = (input.page - 1) * PAGE_SIZE
        recommendations = tuple(
            StoreRecommendation(store_id=store_id, score=score)
            for store_id, score in candidates[offset:]
        )

        self.metrics.increment("recommendation.store_similarity.queries")
        self.logger.info(
            "recommendation.store_similarity.found",
            page=input.page,
            result_count=len(recommendations),
        )

        return FindSimilarStoresOutput(
            recommendations=recommendations,
        )
