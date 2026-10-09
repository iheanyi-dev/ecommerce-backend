"""Structured logging port."""

from typing import Any, Protocol


class Logger(Protocol):
    """Record structured application events."""

    def info(self, event: str, **fields: Any) -> None:
        """Record an informational event."""
        ...

    def warning(self, event: str, **fields: Any) -> None:
        """Record a warning event."""
        ...

    def error(self, event: str, **fields: Any) -> None:
        """Record an error event."""
        ...
