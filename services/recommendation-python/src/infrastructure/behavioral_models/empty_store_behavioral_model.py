"""Empty Store behavioral-model stub."""

from __future__ import annotations

from uuid import UUID


class EmptyStoreBehavioralAssociationModel:
    """Temporary Store behavioral model with no trained model available."""

    def find_associated(
        self,
        store_id: UUID,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        """Return no behavioral associations until the real model exists."""
        return []
