"""DTO for deleting a Store embedding."""

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class DeleteStoreEmbeddingInput:
    """Store deletion request."""

    store_id: UUID
