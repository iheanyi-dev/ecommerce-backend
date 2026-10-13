"""DTOs for semantic Product embedding retrieval."""

from dataclasses import dataclass
from uuid import UUID

from application.dto.scored_product_embedding import ScoredProductEmbedding


@dataclass(frozen=True, slots=True)
class RetrieveProductEmbeddingsInput:
    query: str
    top_k: int = 10
    offset: int = 0
    store_id: UUID | None = None


@dataclass(frozen=True, slots=True)
class RetrieveProductEmbeddingsOutput:
    results: tuple[ScoredProductEmbedding, ...]
