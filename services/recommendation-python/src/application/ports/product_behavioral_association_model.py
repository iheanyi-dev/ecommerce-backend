"""Port for Product behavioral association model inference."""

from __future__ import annotations

from typing import Protocol
from uuid import UUID


class ProductBehavioralAssociationModel(Protocol):
    """Runs inference using the trained Product behavioral model."""

    def find_associated(
        self,
        product_id: UUID,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        """Return associated Product IDs and behavioral association scores.

        The infrastructure implementation owns model loading, caching,
        and inference. The application layer only depends on this
        inference contract.
        """
        ...
