"""Application logging port."""

from __future__ import annotations

from typing import Any, Protocol


class Logger(Protocol):
    """Application-facing structured logging contract."""

    def info(self, event: str, **fields: Any) -> None:
        """Record an informational application event."""

    def warning(self, event: str, **fields: Any) -> None:
        """Record a warning application event."""

    def error(self, event: str, **fields: Any) -> None:
        """Record an error application event."""
