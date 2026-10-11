"""Domain value object for a stored knowledge object's reference."""

from dataclasses import dataclass

from ..errors import EmptyStorageReferenceError


@dataclass(frozen=True, slots=True)
class StorageReference:
    """Identifies the source object in the Storage service."""

    value: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, str) or not self.value.strip():
            raise EmptyStorageReferenceError()

    def __str__(self) -> str:
        return self.value
