"""Port for Store behavioral association model inference."""

from __future__ import annotations

from typing import Protocol
from uuid import UUID


class StoreBehavioralAssociationModel(Protocol):
    """Runs inference using the trained Store behavioral model."""

    def find_associated(
        self,
        store_id: UUID,
        limit: int,
    ) -> list[tuple[UUID, float]]:
        """Return associated Store IDs and behavioral association scores.

        The infrastructure implementation owns model loading, caching,
        and inference. The application layer only depends on this
        inference contract.
        """
        ...
