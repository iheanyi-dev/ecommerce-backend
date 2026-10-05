"""DTOs for deleting a Store embedding."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class DeleteStoreEmbeddingInput:
    """Primitive input identifying the Store embedding to delete."""

    store_id: UUID
