"""Reciprocal Rank Fusion reranker implementation."""

from __future__ import annotations

from application.dto.find_similar_products import ProductRecommendation
from application.dto.find_similar_stores import StoreRecommendation
from application.ports.reranker import Reranker


# Standard RRF uses a constant of 60. Keeping it explicit makes the ranking
# strategy deterministic and gives us a single configuration point if the
# strategy is tuned later.
RRF_RERANKING_K = 60


class RRFReranker(Reranker):
    """Fuse semantic and behavioral recommendation rankings using RRF.

    RRF intentionally ignores the absolute candidate scores. Similarity and
    behavioral models may produce scores on different distributions, making
    direct score averaging unsafe. Their relative ranking positions are the
    reliable common signal at this stage.

    The returned DTO retains the original candidate score. The internal RRF
    score determines ordering only; it does not redefine the meaning of the
    application's ``score`` field.
    """

    def rerank_stores(
        self,
        similar_stores: list[StoreRecommendation],
        associated_stores: list[StoreRecommendation],
    ) -> list[StoreRecommendation]:
        """Fuse semantic and behavioral Store rankings."""
        return self._rerank(
            similar_stores,
            associated_stores,
            key=lambda recommendation: recommendation.store_id,
        )

    def rerank_products(
        self,
        similar_products: list[ProductRecommendation],
        associated_products: list[ProductRecommendation],
    ) -> list[ProductRecommendation]:
        """Fuse semantic and behavioral Product rankings."""
        return self._rerank(
            similar_products,
            associated_products,
            key=lambda recommendation: recommendation.product_id,
        )

    @staticmethod
    def _rerank(
        primary: list,
        secondary: list,
        key,
    ) -> list:
        """Fuse two ranked candidate lists and return them in RRF order."""
        candidates: dict[object, object] = {}
        fused_scores: dict[object, float] = {}
        first_seen: dict[object, int] = {}

        sequence = 0

        for rank, recommendation in enumerate(primary, start=1):
            candidate_id = key(recommendation)

            if candidate_id not in candidates:
                candidates[candidate_id] = recommendation
                first_seen[candidate_id] = sequence
                sequence += 1

            fused_scores[candidate_id] = (
                fused_scores.get(candidate_id, 0.0)
                + 1.0 / (RRF_RERANKING_K + rank)
            )

        for rank, recommendation in enumerate(secondary, start=1):
            candidate_id = key(recommendation)

            if candidate_id not in candidates:
                candidates[candidate_id] = recommendation
                first_seen[candidate_id] = sequence
                sequence += 1

            fused_scores[candidate_id] = (
                fused_scores.get(candidate_id, 0.0)
                + 1.0 / (RRF_RERANKING_K + rank)
            )

        ordered_ids = sorted(
            candidates,
            key=lambda candidate_id: (
                -fused_scores[candidate_id],
                first_seen[candidate_id],
            ),
        )

        return [candidates[candidate_id] for candidate_id in ordered_ids]
