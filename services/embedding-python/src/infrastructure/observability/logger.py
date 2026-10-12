"""Infrastructure implementation of the application logging port."""

from __future__ import annotations

import json
import logging
import sys
from typing import Any, TextIO


class StructuredLogger:
    """JSON structured logger used by the Embedding application layer."""

    def __init__(self, stream: TextIO | None = None) -> None:
        """Create a logger writing structured JSON to the supplied stream."""
        if stream is None:
            stream = sys.stdout

        self._logger = logging.getLogger(
            "embedding.infrastructure.observability"
        )
        self._logger.setLevel(logging.INFO)
        self._logger.propagate = False

        self._handler = logging.StreamHandler(stream)
        self._handler.setFormatter(logging.Formatter("%(message)s"))

        self._logger.handlers.clear()
        self._logger.addHandler(self._handler)

    def _write(self, event: str, level: int, fields: dict[str, Any]) -> None:
        """Write one structured application event."""
        payload = {"event": event, **fields}
        self._logger.log(level, json.dumps(payload, default=str, sort_keys=True))

    def info(self, event: str, **fields: Any) -> None:
        """Record an informational application event."""
        self._write(event, logging.INFO, fields)

    def warning(self, event: str, **fields: Any) -> None:
        """Record a warning application event."""
        self._write(event, logging.WARNING, fields)

    def error(self, event: str, **fields: Any) -> None:
        """Record an error application event."""
        self._write(event, logging.ERROR, fields)


def create_production_logger() -> StructuredLogger:
    """Create the production logger writing structured output to stdout."""
    return StructuredLogger(sys.stdout)
