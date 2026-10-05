"""Domain value object for entity descriptions."""

from __future__ import annotations

from dataclasses import dataclass

from ..errors import EmptyEntityDescriptionError


@dataclass(frozen=True, slots=True)
class EntityDescription:
    """A validated description belonging to a recommendation entity."""

    value: str
    entity_name: str

    def __post_init__(self) -> None:
        if not self.value.strip():
            raise EmptyEntityDescriptionError(self.entity_name)

    def __str__(self) -> str:
        return self.value
