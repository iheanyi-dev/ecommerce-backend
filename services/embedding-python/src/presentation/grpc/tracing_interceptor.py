"""OpenTelemetry server interceptor with W3C trace-context extraction."""

from __future__ import annotations

from collections.abc import Sequence

import grpc
from opentelemetry import trace
from opentelemetry.context import Context
from opentelemetry.trace.propagation.tracecontext import (
    TraceContextTextMapPropagator,
)


class TracingServerInterceptor(grpc.aio.ServerInterceptor):
    """Create a server span and continue an incoming trace when provided."""

    def __init__(self, service_name: str = "embedding") -> None:
        self._tracer = trace.get_tracer(service_name)
        self._propagator = TraceContextTextMapPropagator()

    async def intercept_service(
        self,
        continuation,
        handler_call_details: grpc.aio.HandlerCallDetails,
    ):
        handler = await continuation(handler_call_details)
        if handler is None or handler.unary_unary is None:
            return handler

        metadata = {
            item.key.lower(): item.value
            for item in (handler_call_details.invocation_metadata or ())
        }
        parent_context = self._propagator.extract(metadata)

        async def traced_behavior(request, context):
            method = handler_call_details.method or "embedding.unknown"
            with self._tracer.start_as_current_span(
                method,
                context=parent_context if parent_context is not None else Context(),
                kind=trace.SpanKind.SERVER,
            ) as span:
                span.set_attribute("rpc.system", "grpc")
                span.set_attribute("rpc.method", method)
                return await handler.unary_unary(request, context)

        return grpc.unary_unary_rpc_method_handler(
            traced_behavior,
            request_deserializer=handler.request_deserializer,
            response_serializer=handler.response_serializer,
        )
