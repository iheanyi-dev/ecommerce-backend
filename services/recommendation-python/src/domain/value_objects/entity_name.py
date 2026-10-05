"""Domain value object for entity names."""

from __future__ import annotations

from dataclasses import dataclass

from ..errors import EmptyEntityNameError


@dataclass(frozen=True, slots=True)
class EntityName:
    """A validated name belonging to a recommendation entity."""

    value: str
    entity_name: str

    def __post_init__(self) -> None:
        if not self.value.strip():
            raise EmptyEntityNameError(self.entity_name)

    def __str__(self) -> str:
        return self.value
