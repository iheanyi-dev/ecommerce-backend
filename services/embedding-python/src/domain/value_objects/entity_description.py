"""Domain value object for entity descriptions."""

from dataclasses import dataclass

from ..errors import EmptyEntityDescriptionError


@dataclass(frozen=True, slots=True)
class EntityDescription:
    """A validated entity description."""

    value: str
    entity_name: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, str) or not self.value.strip():
            raise EmptyEntityDescriptionError(self.entity_name)

    def __str__(self) -> str:
        return self.value
