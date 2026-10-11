"""DTOs for semantic Product embedding retrieval."""

from dataclasses import dataclass
from uuid import UUID

from domain.entities import ProductEmbedding


@dataclass(frozen=True, slots=True)
class RetrieveProductEmbeddingsInput:
    query: str
    top_k: int = 10
    offset: int = 0
    store_id: UUID | None = None


@dataclass(frozen=True, slots=True)
class RetrieveProductEmbeddingsOutput:
    results: tuple[ProductEmbedding, ...]
