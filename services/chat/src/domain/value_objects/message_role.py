"""Message role value object."""

from enum import StrEnum

from domain.errors.errors import InvalidMessageRoleError


class MessageRole(StrEnum):
    """Supported roles for chat messages."""

    USER = "user"
    ASSISTANT = "assistant"

    @classmethod
    def from_value(cls, value: str) -> "MessageRole":
        """Create a role from its persisted string representation."""
        try:
            return cls(value)
        except ValueError as exc:
            raise InvalidMessageRoleError(
                f"unsupported message role: {value}"
            ) from exc
