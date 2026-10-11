"""Domain value object for an embedding model version."""

from dataclasses import dataclass

from ..errors import EmptyEmbeddingModelVersionError


@dataclass(frozen=True, slots=True)
class EmbeddingModelVersion:
    """Identifies the model version that generated a vector."""

    value: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, str) or not self.value.strip():
            raise EmptyEmbeddingModelVersionError()

    def __str__(self) -> str:
        return self.value
