"""HTTP middleware for request identity, tracing, and observability."""

from __future__ import annotations

import time
import uuid
from collections.abc import Callable

from fastapi import Request, Response
from opentelemetry import propagate, trace
from opentelemetry.trace.propagation.tracecontext import (
    TraceContextTextMapPropagator,
)
from starlette.middleware.base import BaseHTTPMiddleware


class RequestIDMiddleware(BaseHTTPMiddleware):
    """Ensure every request has a valid request ID."""

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        request_id = request.headers.get("X-Request-ID")

        try:
            request_id = str(uuid.UUID(request_id)) if request_id else str(uuid.uuid4())
        except ValueError:
            request_id = str(uuid.uuid4())

        request.state.request_id = request_id

        return await call_next(request)


class TracingMiddleware(BaseHTTPMiddleware):
    """Extract W3C trace context and create an HTTP request span."""

    def __init__(self, app, service_name: str) -> None:
        super().__init__(app)
        self._service_name = service_name
        self._tracer = trace.get_tracer(service_name)
        self._propagator = TraceContextTextMapPropagator()

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        carrier = {
            key: value
            for key, value in request.headers.items()
        }
        parent_context = self._propagator.extract(carrier)

        route = request.scope.get("route")
        route_path = getattr(route, "path", None)
        span_name = (
            f"{request.method} {route_path}"
            if route_path
            else f"{request.method} recommendation.request"
        )

        with self._tracer.start_as_current_span(
            span_name,
            context=parent_context,
        ):
            return await call_next(request)


class HTTPObservabilityMiddleware(BaseHTTPMiddleware):
    """Record safe HTTP request metrics and structured logs."""

    def __init__(self, app, logger, metrics) -> None:
        super().__init__(app)
        self._logger = logger
        self._metrics = metrics

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        started = time.perf_counter()

        try:
            response = await call_next(request)
        except Exception:
            elapsed = time.perf_counter() - started
            self._metrics.increment("recommendation.http.requests")
            self._metrics.observe("recommendation.http.request.duration", elapsed)
            self._logger.error(
                "recommendation.http.request_failed",
                method=request.method,
                path=request.url.path,
                duration_seconds=elapsed,
            )
            raise

        elapsed = time.perf_counter() - started

        self._metrics.increment("recommendation.http.requests")
        self._metrics.observe("recommendation.http.request.duration", elapsed)

        self._logger.info(
            "recommendation.http.request_completed",
            method=request.method,
            path=request.url.path,
            status_code=response.status_code,
            duration_seconds=elapsed,
        )

        return response
