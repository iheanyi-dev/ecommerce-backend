"""gRPC service-to-service authentication."""

from __future__ import annotations

import grpc


class ServiceAuthInterceptor(grpc.ServerInterceptor):
    """Require the configured service name and secret on every RPC."""

    def __init__(self, service_name: str, service_secret: str) -> None:
        self._service_name = service_name
        self._service_secret = service_secret

    def intercept_service(
        self,
        continuation,
        handler_call_details: grpc.HandlerCallDetails,
    ):
        metadata = dict(handler_call_details.invocation_metadata or [])

        if (
            metadata.get("x-service-name") != self._service_name
            or metadata.get("x-service-secret") != self._service_secret
        ):
            return grpc.unary_unary_rpc_method_handler(
                self._unauthorized,
            )

        return continuation(handler_call_details)

    @staticmethod
    def _unauthorized(request, context):
        context.abort(
            grpc.StatusCode.UNAUTHENTICATED,
            "service authentication failed",
        )
