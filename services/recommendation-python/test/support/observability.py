"""Test doubles for application observability ports."""

from __future__ import annotations

from typing import Any


class StubLogger:
    """No-op logger for application use-case tests."""

    def info(self, event: str, **fields: Any) -> None:
        pass

    def warning(self, event: str, **fields: Any) -> None:
        pass

    def error(self, event: str, **fields: Any) -> None:
        pass


class StubMetrics:
    """No-op metrics collector for application use-case tests."""

    def increment(self, name: str, value: int = 1) -> None:
        pass

    def observe(self, name: str, value: float) -> None:
        pass
