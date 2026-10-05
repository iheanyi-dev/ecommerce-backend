"""Use case for finding Products associated by user behavior."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_associated_products import (
    FindAssociatedProductsInput,
    FindAssociatedProductsOutput,
)
from ..dto.find_similar_products import ProductRecommendation
from ..ports.product_behavioral_association_model import (
    ProductBehavioralAssociationModel,
)


PAGE_SIZE = 10


@dataclass(slots=True)
class FindAssociatedProducts:
    """Find Product recommendations from behavioral model inference.

    The trained model is an inference dependency, not a persistence
    repository. Model loading, caching, and inference remain infrastructure
    responsibilities.
    """

    product_behavioral_association_model: ProductBehavioralAssociationModel

    def execute(
        self,
        input: FindAssociatedProductsInput,
    ) -> FindAssociatedProductsOutput:
        """Run behavioral inference and return the requested page."""

        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        candidate_limit = input.page * PAGE_SIZE

        candidates = (
            self.product_behavioral_association_model.find_associated(
                product_id=input.product_id,
                limit=candidate_limit,
            )
        )

        offset = (input.page - 1) * PAGE_SIZE

        recommendations = tuple(
            ProductRecommendation(
                product_id=product_id,
                score=score,
            )
            for product_id, score in candidates[offset:]
        )

        return FindAssociatedProductsOutput(
            recommendations=recommendations,
        )
