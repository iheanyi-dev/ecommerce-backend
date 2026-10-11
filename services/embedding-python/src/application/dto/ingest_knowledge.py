"""DTOs for ingesting a knowledge document."""

from dataclasses import dataclass
from uuid import UUID

from domain.value_objects import EmbeddingModelVersion


@dataclass(frozen=True, slots=True)
class IngestKnowledgeInput:
    """Identify a knowledge document and its stored source."""

    knowledge_id: UUID
    store_id: UUID
    storage_reference: str


@dataclass(frozen=True, slots=True)
class IngestKnowledgeOutput:
    """Summary of a successfully ingested knowledge document."""

    knowledge_id: UUID
    store_id: UUID
    chunk_count: int
    embedding_model_versions: tuple[EmbeddingModelVersion, ...]
