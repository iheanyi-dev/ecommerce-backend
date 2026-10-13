"""Use case for finding products similar to a query."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_similar_products import (
    FindSimilarProductsInput,
    FindSimilarProductsOutput,
    ProductRecommendation,
)
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.similarity_retriever import SimilarityRetriever

PAGE_SIZE = 10


@dataclass(slots=True)
class FindSimilarProducts:
    """Find semantic product candidates through Embedding."""

    similarity_retriever: SimilarityRetriever
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: FindSimilarProductsInput,
    ) -> FindSimilarProductsOutput:
        """Retrieve the requested page of similar products."""
        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        recommendations = self.similarity_retriever.find_similar_products(
            query=input.query,
            top_k=PAGE_SIZE,
            offset=(input.page - 1) * PAGE_SIZE,
        )
        output = FindSimilarProductsOutput(
            recommendations=tuple(
                ProductRecommendation(product_id=product_id, score=score)
                for product_id, score in recommendations
            )
        )

        self.metrics.increment("recommendation.product_similarity.queries")
        self.logger.info(
            "recommendation.product_similarity.found",
            page=input.page,
            result_count=len(output.recommendations),
        )
        return output
