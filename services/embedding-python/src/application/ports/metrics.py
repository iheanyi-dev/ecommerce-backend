"""Application metrics port."""

from typing import Protocol


class Metrics(Protocol):
    """Application-facing metrics contract."""

    def increment(self, name: str, value: int = 1) -> None:
        """Increment a counter."""
        ...

    def observe(self, name: str, value: float) -> None:
        """Record an observed numeric value."""
        ...
