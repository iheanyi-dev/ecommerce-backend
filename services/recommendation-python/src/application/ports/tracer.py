"""Application tracing port."""

from __future__ import annotations

from typing import Any, Protocol


class Span(Protocol):
    """Minimal tracing span contract exposed to the application layer."""

    def end(self) -> None:
        """End the tracing span."""


class Tracer(Protocol):
    """Application-facing tracing contract."""

    def start(self, context: Any, name: str) -> tuple[Any, Span]:
        """Create a span and return its propagated context."""
