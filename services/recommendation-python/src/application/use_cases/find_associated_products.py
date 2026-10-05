"""Use case for finding Products associated by user behavior."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_associated_products import (
    FindAssociatedProductsInput,
    FindAssociatedProductsOutput,
)
from ..dto.find_similar_products import ProductRecommendation
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.product_behavioral_association_model import (
    ProductBehavioralAssociationModel,
)

PAGE_SIZE = 10


@dataclass(slots=True)
class FindAssociatedProducts:
    """Find Product recommendations from behavioral model inference."""

    product_behavioral_association_model: ProductBehavioralAssociationModel
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: FindAssociatedProductsInput,
    ) -> FindAssociatedProductsOutput:
        """Run behavioral inference and return the requested page."""

        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        candidates = self.product_behavioral_association_model.find_associated(
            product_id=input.product_id,
            limit=input.page * PAGE_SIZE,
        )

        offset = (input.page - 1) * PAGE_SIZE
        recommendations = tuple(
            ProductRecommendation(product_id=product_id, score=score)
            for product_id, score in candidates[offset:]
        )

        self.metrics.increment(
            "recommendation.product_associations.queries",
        )
        self.logger.info(
            "recommendation.product_associations.found",
            product_id=str(input.product_id),
            page=input.page,
            result_count=len(recommendations),
        )

        return FindAssociatedProductsOutput(
            recommendations=recommendations,
        )
