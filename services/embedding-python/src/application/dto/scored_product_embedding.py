"""A Product embedding and its query-specific similarity score."""

from dataclasses import dataclass

from domain.entities.product_embedding import ProductEmbedding


@dataclass(frozen=True, slots=True)
class ScoredProductEmbedding:
    """Pair a persisted Product embedding with its Qdrant similarity score."""

    embedding: ProductEmbedding
    score: float
