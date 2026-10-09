"""Knowledge domain entity."""

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True, slots=True)
class Knowledge:
    """A knowledge resource belonging to a store."""

    id: UUID
    store_id: UUID
    filename: str
    content_type: str

    @property
    def storage_reference(self) -> str:
        """Return the deterministic storage reference for the knowledge resource."""
        return f"knowledge/{self.store_id}/{self.filename}"
