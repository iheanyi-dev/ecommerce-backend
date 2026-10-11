"""Domain value object for entity identifiers."""

from dataclasses import dataclass
from uuid import UUID

from ..errors import InvalidEntityIDError


@dataclass(frozen=True, slots=True)
class EntityID:
    """A validated UUID identifying a domain entity."""

    value: UUID
    entity_name: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, UUID):
            raise InvalidEntityIDError(self.entity_name)

    def __str__(self) -> str:
        return str(self.value)
