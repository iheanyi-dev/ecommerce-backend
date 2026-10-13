"""A Store embedding and its query-specific similarity score."""

from dataclasses import dataclass

from domain.entities.store_embedding import StoreEmbedding


@dataclass(frozen=True, slots=True)
class ScoredStoreEmbedding:
    """Pair a persisted Store embedding with its Qdrant similarity score."""

    embedding: StoreEmbedding
    score: float
