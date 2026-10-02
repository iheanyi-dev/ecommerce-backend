"""Domain value object representing an embedding model version."""

from __future__ import annotations

from dataclasses import dataclass

from ..errors import EmptyEmbeddingModelVersionError


@dataclass(frozen=True, slots=True)
class EmbeddingModelVersion:
    """
    Immutable identifier for the model version that generated an embedding.

    Keeping the model version with the representation allows the service to
    identify stale embeddings when a newer model becomes active.
    """

    value: str

    def __post_init__(self) -> None:
        """Validate the model version."""
        if not self.value.strip():
            raise EmptyEmbeddingModelVersionError()

    def __str__(self) -> str:
        """Return the model version value."""
        return self.value
