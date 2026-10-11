"""Domain value object for knowledge chunk content."""

from dataclasses import dataclass

from ..errors import EmptyKnowledgeChunkContentError


@dataclass(frozen=True, slots=True)
class KnowledgeChunkContent:
    """Validated text belonging to a knowledge chunk."""

    value: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, str) or not self.value.strip():
            raise EmptyKnowledgeChunkContentError()

    def __str__(self) -> str:
        return self.value
