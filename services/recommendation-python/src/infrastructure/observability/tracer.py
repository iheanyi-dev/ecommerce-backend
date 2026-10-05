"""OpenTelemetry infrastructure implementation of the tracing port."""

from __future__ import annotations

from typing import Any

from opentelemetry import propagate, trace
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

from application.ports.tracer import Span


class _NoopSpan:
    """Safe no-op span used when tracing is unavailable or invalid."""

    def end(self) -> None:
        """End the no-op span."""
        return


class _OpenTelemetrySpan:
    """Adapter from an OpenTelemetry span to the application Span port."""

    def __init__(self, span: trace.Span) -> None:
        """Store the concrete OpenTelemetry span."""
        self._span = span

    def end(self) -> None:
        """End the underlying OpenTelemetry span."""
        self._span.end()


class OpenTelemetryTracer:
    """Infrastructure implementation of the application Tracer port."""

    def __init__(self, tracer: trace.Tracer) -> None:
        """Create an adapter around an OpenTelemetry tracer."""
        self._tracer = tracer

    @classmethod
    def create(cls, service_name: str) -> OpenTelemetryTracer:
        """Create a service tracer and configure W3C trace propagation."""
        service_name = service_name.strip()

        if not service_name:
            raise ValueError("tracer name cannot be empty")

        propagate.set_global_textmap(TraceContextTextMapPropagator())

        return cls(trace.get_tracer(service_name))

    def start(
        self,
        context: Any,
        name: str,
    ) -> tuple[Any, Span]:
        """Create a span while preserving the supplied trace context."""
        if self._tracer is None:
            return context, _NoopSpan()

        name = name.strip()

        if not name:
            return context, _NoopSpan()

        span_context = (
            context if context is not None else trace.context_api.Context()
        )

        span = self._tracer.start_span(
            name,
            context=span_context,
        )

        propagated_context = trace.set_span_in_context(
            span,
            span_context,
        )

        return propagated_context, _OpenTelemetrySpan(span)


def create_production_tracer(service_name: str) -> OpenTelemetryTracer:
    """Create the production Recommendation Service tracer."""
    return OpenTelemetryTracer.create(service_name)
