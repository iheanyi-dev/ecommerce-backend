"""Use case for finding Stores associated by user behavior."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_associated_stores import (
    FindAssociatedStoresInput,
    FindAssociatedStoresOutput,
)
from ..dto.find_similar_stores import StoreRecommendation
from ..ports.logger import Logger
from ..ports.metrics import Metrics
from ..ports.store_behavioral_association_model import (
    StoreBehavioralAssociationModel,
)

PAGE_SIZE = 10


@dataclass(slots=True)
class FindAssociatedStores:
    """Find Store recommendations from behavioral model inference."""

    store_behavioral_association_model: StoreBehavioralAssociationModel
    logger: Logger
    metrics: Metrics

    def execute(
        self,
        input: FindAssociatedStoresInput,
    ) -> FindAssociatedStoresOutput:
        """Run behavioral inference and return the requested page."""

        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        candidates = self.store_behavioral_association_model.find_associated(
            store_id=input.store_id,
            limit=input.page * PAGE_SIZE,
        )

        offset = (input.page - 1) * PAGE_SIZE
        recommendations = tuple(
            StoreRecommendation(store_id=store_id, score=score)
            for store_id, score in candidates[offset:]
        )

        self.metrics.increment(
            "recommendation.store_associations.queries",
        )
        self.logger.info(
            "recommendation.store_associations.found",
            store_id=str(input.store_id),
            page=input.page,
            result_count=len(recommendations),
        )

        return FindAssociatedStoresOutput(
            recommendations=recommendations,
        )
