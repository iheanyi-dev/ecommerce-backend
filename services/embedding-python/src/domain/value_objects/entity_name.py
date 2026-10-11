"""Domain value object for entity names."""

from dataclasses import dataclass

from ..errors import EmptyEntityNameError


@dataclass(frozen=True, slots=True)
class EntityName:
    """A validated entity name."""

    value: str
    entity_name: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, str) or not self.value.strip():
            raise EmptyEntityNameError(self.entity_name)

    def __str__(self) -> str:
        return self.value
