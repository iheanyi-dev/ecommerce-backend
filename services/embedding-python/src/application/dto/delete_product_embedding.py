"""DTO for deleting a Product embedding."""

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class DeleteProductEmbeddingInput:
    """Product deletion request."""

    product_id: UUID
