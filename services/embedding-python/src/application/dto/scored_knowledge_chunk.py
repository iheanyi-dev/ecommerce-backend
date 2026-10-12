"""Application result carrying a chunk and its query-specific similarity score."""

from dataclasses import dataclass

from domain.entities.knowledge_chunk import KnowledgeChunk


@dataclass(frozen=True, slots=True)
class ScoredKnowledgeChunk:
    """Represent a retrieved chunk without persisting its query-specific score."""

    chunk: KnowledgeChunk
    score: float
