"""Tests for the infrastructure structured logger."""

from __future__ import annotations

import io
import json

from infrastructure.observability.logger import StructuredLogger


def test_logger_writes_structured_json() -> None:
    stream = io.StringIO()
    logger = StructuredLogger(stream)

    logger.info(
        "recommendation.stores.completed",
        result_count=10,
        query_provided=True,
    )

    payload = json.loads(stream.getvalue())

    assert payload == {
        "event": "recommendation.stores.completed",
        "query_provided": True,
        "result_count": 10,
    }


def test_logger_supports_warning_and_error_events() -> None:
    stream = io.StringIO()
    logger = StructuredLogger(stream)

    logger.warning("recommendation.warning", reason="test")
    logger.error("recommendation.error", reason="test")

    lines = stream.getvalue().splitlines()

    assert json.loads(lines[0]) == {
        "event": "recommendation.warning",
        "reason": "test",
    }
    assert json.loads(lines[1]) == {
        "event": "recommendation.error",
        "reason": "test",
    }


def test_logger_does_not_invent_sensitive_fields() -> None:
    stream = io.StringIO()
    logger = StructuredLogger(stream)

    logger.info("recommendation.test", result_count=1)

    payload = json.loads(stream.getvalue())

    assert "password" not in payload
    assert "token" not in payload
    assert "authorization" not in payload
