"""Use case for finding Stores associated by user behavior."""

from __future__ import annotations

from dataclasses import dataclass

from ..dto.find_associated_stores import (
    FindAssociatedStoresInput,
    FindAssociatedStoresOutput,
)
from ..dto.find_similar_stores import StoreRecommendation
from ..ports.store_behavioral_association_model import (
    StoreBehavioralAssociationModel,
)


PAGE_SIZE = 10


@dataclass(slots=True)
class FindAssociatedStores:
    """Find Store recommendations from behavioral model inference.

    The trained model is an inference dependency, not a persistence
    repository. Model loading, caching, and inference remain infrastructure
    responsibilities.
    """

    store_behavioral_association_model: StoreBehavioralAssociationModel

    def execute(
        self,
        input: FindAssociatedStoresInput,
    ) -> FindAssociatedStoresOutput:
        """Run behavioral inference and return the requested page."""

        if input.page < 1:
            raise ValueError("page must be greater than or equal to 1")

        candidate_limit = input.page * PAGE_SIZE

        candidates = (
            self.store_behavioral_association_model.find_associated(
                store_id=input.store_id,
                limit=candidate_limit,
            )
        )

        offset = (input.page - 1) * PAGE_SIZE

        recommendations = tuple(
            StoreRecommendation(
                store_id=store_id,
                score=score,
            )
            for store_id, score in candidates[offset:]
        )

        return FindAssociatedStoresOutput(
            recommendations=recommendations,
        )
