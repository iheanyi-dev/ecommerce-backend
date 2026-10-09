"""Application metrics port."""

from typing import Protocol


class Metrics(Protocol):
    """Record counters and numeric observations."""

    def increment(self, name: str, value: int = 1) -> None:
        """Increment a named counter."""
        ...

    def observe(self, name: str, value: float) -> None:
        """Record a numeric observation."""
        ...
