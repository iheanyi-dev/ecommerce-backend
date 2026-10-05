"""Tests for the OpenTelemetry tracing infrastructure adapter."""

from __future__ import annotations

from opentelemetry import propagate, trace
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
from opentelemetry.sdk.trace import TracerProvider

import pytest

from infrastructure.observability.tracer import OpenTelemetryTracer


def test_create_rejects_empty_service_name() -> None:
    with pytest.raises(ValueError, match="tracer name cannot be empty"):
        OpenTelemetryTracer.create("   ")


def test_create_returns_tracer() -> None:
    tracer = OpenTelemetryTracer.create("recommendation")

    assert tracer is not None


def test_start_creates_span() -> None:
    provider = TracerProvider()
    previous_provider = trace.get_tracer_provider()

    trace.set_tracer_provider(provider)

    try:
        tracer = OpenTelemetryTracer.create("recommendation")

        context, span = tracer.start(
            trace.context_api.Context(),
            "recommendation.test",
        )

        span_from_context = trace.get_current_span(context)

        assert context is not None
        assert span is not None
        assert span_from_context.get_span_context().is_valid

        span.end()
    finally:
        trace.set_tracer_provider(previous_provider)
        provider.shutdown()


def test_start_with_empty_span_name_does_not_create_span() -> None:
    provider = TracerProvider()
    previous_provider = trace.get_tracer_provider()

    trace.set_tracer_provider(provider)

    try:
        tracer = OpenTelemetryTracer.create("recommendation")

        original_context = trace.context_api.Context()

        context, span = tracer.start(
            original_context,
            "   ",
        )

        assert context is original_context
        assert span is not None
        assert not trace.get_current_span(context).get_span_context().is_valid

        span.end()
    finally:
        trace.set_tracer_provider(previous_provider)
        provider.shutdown()


def test_start_preserves_parent_trace() -> None:
    provider = TracerProvider()
    previous_provider = trace.get_tracer_provider()

    trace.set_tracer_provider(provider)

    try:
        tracer = OpenTelemetryTracer.create("recommendation")

        parent_context, parent_span = tracer.start(
            trace.context_api.Context(),
            "recommendation.parent",
        )

        child_context, child_span = tracer.start(
            parent_context,
            "recommendation.child",
        )

        parent_span_context = trace.get_current_span(
            parent_context,
        ).get_span_context()

        child_span_context = trace.get_current_span(
            child_context,
        ).get_span_context()

        assert parent_span_context.is_valid
        assert child_span_context.is_valid
        assert parent_span_context.trace_id == child_span_context.trace_id

        parent_span.end()
        child_span.end()
    finally:
        trace.set_tracer_provider(previous_provider)
        provider.shutdown()


def test_create_configures_w3c_trace_context_propagator() -> None:
    previous_propagator = propagate.get_global_textmap()

    try:
        OpenTelemetryTracer.create("recommendation")

        assert isinstance(
            propagate.get_global_textmap(),
            TraceContextTextMapPropagator,
        )
    finally:
        propagate.set_global_textmap(previous_propagator)


def test_noop_span_is_safe_when_tracer_is_unavailable() -> None:
    tracer = OpenTelemetryTracer.__new__(OpenTelemetryTracer)
    tracer._tracer = None

    context = trace.context_api.Context()

    returned_context, span = tracer.start(
        context,
        "recommendation.test",
    )

    assert returned_context is context
    assert span is not None

    span.end()
