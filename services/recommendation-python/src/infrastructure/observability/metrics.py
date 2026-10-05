"""Infrastructure implementation of the application metrics port."""

from __future__ import annotations

from threading import RLock


class Metrics:
    """Thread-safe in-memory metrics collector.

    The application layer records metrics through its Protocol. Keeping the
    concrete implementation in Infrastructure avoids coupling application
    code to Prometheus, OpenTelemetry, or another monitoring backend.
    """

    def __init__(self) -> None:
        """Create an empty metrics collector."""
        self._lock = RLock()
        self._counters: dict[str, float] = {}
        self._observations: dict[str, list[float]] = {}

    def increment(self, name: str, value: int = 1) -> None:
        """Increment a counter metric."""
        if not name.strip():
            raise ValueError("metric name cannot be empty")

        with self._lock:
            self._counters[name] = self._counters.get(name, 0.0) + value

    def observe(self, name: str, value: float) -> None:
        """Record a numeric metric observation."""
        if not name.strip():
            raise ValueError("metric name cannot be empty")

        with self._lock:
            self._observations.setdefault(name, []).append(value)

    def counter_value(self, name: str) -> float:
        """Return the current counter value."""
        with self._lock:
            return self._counters.get(name, 0.0)

    def observation_values(self, name: str) -> list[float]:
        """Return a copy of the observations for a metric."""
        with self._lock:
            return list(self._observations.get(name, []))
