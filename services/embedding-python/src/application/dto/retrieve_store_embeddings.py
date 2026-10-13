"""DTOs for semantic Store embedding retrieval."""

from dataclasses import dataclass

from application.dto.scored_store_embedding import ScoredStoreEmbedding


@dataclass(frozen=True, slots=True)
class RetrieveStoreEmbeddingsInput:
    query: str
    top_k: int = 10
    offset: int = 0


@dataclass(frozen=True, slots=True)
class RetrieveStoreEmbeddingsOutput:
    results: tuple[ScoredStoreEmbedding, ...]
