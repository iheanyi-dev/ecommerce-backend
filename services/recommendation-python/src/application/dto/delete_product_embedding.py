"""DTOs for deleting a Product embedding."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class DeleteProductEmbeddingInput:
    """Primitive input identifying the Product embedding to delete."""

    product_id: UUID
