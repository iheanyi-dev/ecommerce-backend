"""Tests for the infrastructure structured logger."""

from __future__ import annotations

import io
import json

from infrastructure.observability.logger import StructuredLogger


def test_logger_writes_structured_json() -> None:
    stream = io.StringIO()
    logger = StructuredLogger(stream)

    logger.info(
        "embedding.knowledge.ingestion.completed",
        chunk_count=10,
        knowledge_id="knowledge-123",
    )

    assert json.loads(stream.getvalue()) == {
        "event": "embedding.knowledge.ingestion.completed",
        "chunk_count": 10,
        "knowledge_id": "knowledge-123",
    }


def test_logger_supports_warning_and_error_events() -> None:
    stream = io.StringIO()
    logger = StructuredLogger(stream)

    logger.warning("embedding.warning", reason="test")
    logger.error("embedding.error", reason="test")

    lines = stream.getvalue().splitlines()

    assert json.loads(lines[0]) == {
        "event": "embedding.warning",
        "reason": "test",
    }
    assert json.loads(lines[1]) == {
        "event": "embedding.error",
        "reason": "test",
    }


def test_logger_does_not_invent_sensitive_fields() -> None:
    stream = io.StringIO()
    logger = StructuredLogger(stream)

    logger.info("embedding.test", chunk_count=1)

    payload = json.loads(stream.getvalue())

    assert "password" not in payload
    assert "token" not in payload
    assert "authorization" not in payload
