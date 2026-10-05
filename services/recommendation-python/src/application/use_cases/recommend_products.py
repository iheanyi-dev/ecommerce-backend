"""External use case for Product recommendations."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_associated_products import FindAssociatedProductsInput
from ..dto.find_similar_products import FindSimilarProductsInput
from ..dto.recommend_products import (
    RecommendProductsInput,
    RecommendProductsOutput,
)
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.reranker import Reranker
from .find_associated_products import FindAssociatedProducts
from .find_similar_products import FindSimilarProducts


@dataclass(slots=True)
class RecommendProducts:
    """Orchestrate Product candidate generation and reranking."""

    find_similar_products: FindSimilarProducts
    find_associated_products: FindAssociatedProducts
    reranker: Reranker
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: RecommendProductsInput,
    ) -> RecommendProductsOutput:
        """Generate available candidates and return their final ranking."""

        similar_products = []

        if input.query is not None:
            result = self.find_similar_products.execute(
                FindSimilarProductsInput(query=input.query, page=1)
            )
            similar_products = list(result.recommendations)

        associated_products = []

        if input.product_id is not None:
            result = self.find_associated_products.execute(
                FindAssociatedProductsInput(
                    product_id=input.product_id,
                    page=1,
                )
            )
            associated_products = list(result.recommendations)

        recommendations = self.reranker.rerank_products(
            similar_products=similar_products,
            associated_products=associated_products,
        )

        self.metrics.increment("recommendation.product.queries")
        self.logger.info(
            "recommendation.products.completed",
            query_provided=input.query is not None,
            product_id_provided=input.product_id is not None,
            result_count=len(recommendations),
        )

        return RecommendProductsOutput(
            recommendations=tuple(recommendations),
        )
