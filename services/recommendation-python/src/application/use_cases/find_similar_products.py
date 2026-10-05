"""Use case for finding Products similar to a query."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_similar_products import (
    FindSimilarProductsInput,
    FindSimilarProductsOutput,
    ProductRecommendation,
)
from ..ports.embedding_generator import EmbeddingGenerator
from ..ports.product_embedding_repository import ProductEmbeddingRepository


PAGE_SIZE = 10


@dataclass(slots=True)
class FindSimilarProducts:
    """Find Product recommendations for a text query.

    Pagination is deliberately owned by the application layer. The client
    supplies only the requested page; the recommendation page size remains
    an application policy.
    """

    embedding_generator: EmbeddingGenerator
    product_embedding_repository: ProductEmbeddingRepository

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

        candidate_limit = input.page * PAGE_SIZE

        candidates = self.product_embedding_repository.find_similar(
            embedding=query_embedding,
            limit=candidate_limit,
        )

        offset = (input.page - 1) * PAGE_SIZE

        recommendations = tuple(
            ProductRecommendation(
                product_id=product_id,
                score=score,
            )
            for product_id, score in candidates[offset:]
        )

        return FindSimilarProductsOutput(
            recommendations=recommendations,
        )
