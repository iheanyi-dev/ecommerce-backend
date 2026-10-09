"""Value object representing validated chat message content."""

from dataclasses import dataclass

from domain.errors.errors import InvalidMessageContentError


@dataclass(frozen=True, slots=True)
class MessageContent:
    """Validated message content."""

    value: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, str) or not self.value.strip():
            raise InvalidMessageContentError(
                "message content cannot be empty"
            )
