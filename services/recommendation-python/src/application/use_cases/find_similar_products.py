"""Use case for finding Products similar to a query."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_similar_products import (
    FindSimilarProductsInput,
    FindSimilarProductsOutput,
    ProductRecommendation,
)
from ..ports.embedding_generator import EmbeddingGenerator
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.product_embedding_repository import ProductEmbeddingRepository

PAGE_SIZE = 10


@dataclass(slots=True)
class FindSimilarProducts:
    """Find Product recommendations for a text query."""

    embedding_generator: EmbeddingGenerator
    product_embedding_repository: ProductEmbeddingRepository
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: FindSimilarProductsInput,
    ) -> FindSimilarProductsOutput:
        """Generate the query embedding and return the requested page."""

        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        query_embedding = (
            self.embedding_generator.generate_product_query_embedding(
                input.query,
            )
        )

        candidates = self.product_embedding_repository.find_similar(
            embedding=query_embedding,
            limit=input.page * PAGE_SIZE,
        )

        offset = (input.page - 1) * PAGE_SIZE
        recommendations = tuple(
            ProductRecommendation(product_id=product_id, score=score)
            for product_id, score in candidates[offset:]
        )

        self.metrics.increment("recommendation.product_similarity.queries")
        self.logger.info(
            "recommendation.product_similarity.found",
            page=input.page,
            result_count=len(recommendations),
        )

        return FindSimilarProductsOutput(
            recommendations=recommendations,
        )
