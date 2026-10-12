"""Tests for the infrastructure metrics collector."""

from __future__ import annotations

import threading

import pytest

from infrastructure.observability.metrics import Metrics


def test_metrics_increment_records_counter() -> None:
    metrics = Metrics()

    metrics.increment("embedding.knowledge.ingestions")

    assert metrics.counter_value("embedding.knowledge.ingestions") == 1


def test_metrics_increment_accumulates_values() -> None:
    metrics = Metrics()

    metrics.increment("embedding.knowledge.ingestions")
    metrics.increment("embedding.knowledge.ingestions", 4)

    assert metrics.counter_value("embedding.knowledge.ingestions") == 5


def test_metrics_observe_records_measurements() -> None:
    metrics = Metrics()

    metrics.observe("embedding.ingestion.duration", 12.5)
    metrics.observe("embedding.ingestion.duration", 18.25)

    assert metrics.observation_values("embedding.ingestion.duration") == [
        12.5,
        18.25,
    ]


def test_metrics_rejects_empty_counter_name() -> None:
    metrics = Metrics()

    with pytest.raises(ValueError, match="metric name cannot be empty"):
        metrics.increment(" ")


def test_metrics_rejects_empty_observation_name() -> None:
    metrics = Metrics()

    with pytest.raises(ValueError, match="metric name cannot be empty"):
        metrics.observe("", 1.0)


def test_metrics_returns_copy_of_observations() -> None:
    metrics = Metrics()
    metrics.observe("embedding.ingestion.duration", 10.0)

    values = metrics.observation_values("embedding.ingestion.duration")
    values.append(99.0)

    assert metrics.observation_values("embedding.ingestion.duration") == [10.0]


def test_metrics_is_safe_for_concurrent_counter_updates() -> None:
    metrics = Metrics()

    def increment() -> None:
        for _ in range(100):
            metrics.increment("embedding.concurrent")

    threads = [threading.Thread(target=increment) for _ in range(10)]

    for thread in threads:
        thread.start()

    for thread in threads:
        thread.join()

    assert metrics.counter_value("embedding.concurrent") == 1000
