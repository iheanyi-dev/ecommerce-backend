"""Empty Product behavioral-model stub."""

from __future__ import annotations

from uuid import UUID


class EmptyProductBehavioralAssociationModel:
    """Temporary Product behavioral model with no trained model available."""

    def find_associated(
        self,
        product_id: UUID,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        """Return no behavioral associations until the real model exists."""
        return []
