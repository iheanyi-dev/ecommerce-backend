"""Domain value object for a knowledge chunk's position."""

from dataclasses import dataclass

from ..errors import InvalidChunkIndexError


@dataclass(frozen=True, slots=True)
class ChunkIndex:
    """Zero-based position of a chunk within its source knowledge."""

    value: int

    def __post_init__(self) -> None:
        if isinstance(self.value, bool) or not isinstance(self.value, int):
            raise InvalidChunkIndexError()
        if self.value < 0:
            raise InvalidChunkIndexError()
