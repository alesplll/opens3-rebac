"""
gRPC interceptors for distributed tracing — Python analogue of go-kit/tracing/grpc_interceptor.go.

Provides:
  - TracingServerInterceptor  — extracts W3C trace context from incoming gRPC metadata,
                                starts a server span, injects trace_id into response metadata.
  - TracingClientInterceptor  — starts a client span and injects trace context into outgoing metadata.

Usage (server):
    from shared.pkg.py_kit.tracing.grpc_interceptor import TracingServerInterceptor
    from shared.pkg.py_kit.tracing import init_tracer

    init_tracer(cfg)
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        interceptors=[TracingServerInterceptor("authz")],
    )

Usage (client stub — when calling another gRPC service):
    from shared.pkg.py_kit.tracing.grpc_interceptor import make_tracing_client_interceptor

    channel = grpc.intercept_channel(
        grpc.insecure_channel("metadata:50052"),
        make_tracing_client_interceptor("authz"),
    )
"""

from __future__ import annotations

import logging
from typing import Any, Callable

import grpc

log = logging.getLogger(__name__)

TRACE_ID_HEADER = "x-trace_id"  # mirrors Go's tracing.TraceIDHeader


# ---------------------------------------------------------------------------
# Metadata carrier — mirrors go-kit tracing/metadata_carrier.go
# ---------------------------------------------------------------------------

class _MetadataCarrier:
    """Adapts gRPC metadata to the OTel TextMapCarrier interface."""

    def __init__(self, metadata: grpc.aio.Metadata | list | None = None) -> None:
        # Store as a mutable dict of header -> list[str]
        self._data: dict[str, list[str]] = {}
        if metadata:
            for k, v in metadata:
                key = k.lower()
                self._data.setdefault(key, []).append(v)

    def get(self, key: str) -> str:
        values = self._data.get(key.lower(), [])
        return values[0] if values else ""

    def set(self, key: str, value: str) -> None:
        self._data[key.lower()] = [value]

    def keys(self) -> list[str]:
        return list(self._data.keys())

    def to_list(self) -> list[tuple[str, str]]:
        """Convert back to gRPC metadata list format."""
        result = []
        for k, vals in self._data.items():
            for v in vals:
                result.append((k, v))
        return result


# ---------------------------------------------------------------------------
# Server interceptor
# ---------------------------------------------------------------------------

class TracingServerInterceptor(grpc.ServerInterceptor):
    """
    gRPC server interceptor that:
    1. Extracts W3C trace context from incoming request metadata.
    2. Starts a server-side span named after the RPC method.
    3. Records errors on the span.
    4. Adds x-trace_id to outgoing response metadata.

    Mirrors Go's tracing.UnaryServerInterceptor(serviceName).
    """

    def __init__(self, service_name: str) -> None:
        self._service_name = service_name
        self._otel_available = self._check_otel()

    @staticmethod
    def _check_otel() -> bool:
        try:
            import opentelemetry.trace  # noqa: F401
            return True
        except ImportError:
            return False

    def intercept_service(
        self,
        continuation: Callable,
        handler_call_details: grpc.HandlerCallDetails,
    ):
        handler = continuation(handler_call_details)
        if handler is None or not self._otel_available:
            return handler

        method = handler_call_details.method  # e.g. /rebac.authz.v1.PermissionService/Check

        def wrap(behavior: Callable):
            def new_behavior(request: Any, context: grpc.ServicerContext) -> Any:
                self._start_server_span(method, context)
                try:
                    result = behavior(request, context)
                    return result
                except Exception as exc:
                    self._record_error(exc)
                    raise

            return new_behavior

        # Only wrap unary_unary — extend for streaming if needed
        if handler.unary_unary:
            return handler._replace(unary_unary=wrap(handler.unary_unary))
        return handler

    def _start_server_span(self, method: str, context: grpc.ServicerContext) -> None:
        try:
            from opentelemetry import trace
            from opentelemetry.propagate import get_global_textmap

            # Extract trace context from incoming gRPC metadata
            carrier = _MetadataCarrier(context.invocation_metadata())
            ctx = get_global_textmap().extract(carrier)

            tracer = trace.get_tracer(self._service_name)
            span = tracer.start_span(
                method,
                context=ctx,
                kind=trace.SpanKind.SERVER,
            )

            # Attach span to current context (OTel contextvars)
            token = trace.use_span(span, end_on_exit=True)  # type: ignore[attr-defined]

            # Inject trace_id into response metadata
            span_ctx = span.get_span_context()
            if span_ctx and span_ctx.is_valid:
                trace_id_hex = format(span_ctx.trace_id, "032x")
                try:
                    context.send_initial_metadata([(TRACE_ID_HEADER, trace_id_hex)])
                except Exception:
                    pass  # metadata already sent or not supported

        except Exception as exc:
            log.debug("TracingServerInterceptor: failed to start span: %s", exc)

    def _record_error(self, exc: Exception) -> None:
        try:
            from opentelemetry import trace
            span = trace.get_current_span()
            if span:
                span.record_exception(exc)
        except Exception:
            pass


# ---------------------------------------------------------------------------
# Client interceptor
# ---------------------------------------------------------------------------

class _TracingClientInterceptor(
    grpc.UnaryUnaryClientInterceptor,
    grpc.UnaryStreamClientInterceptor,
):
    """
    gRPC client interceptor that injects the current trace context into
    outgoing request metadata. Mirrors Go's tracing.UnaryClientInterceptor().
    """

    def __init__(self, service_name: str) -> None:
        self._service_name = service_name

    def intercept_unary_unary(self, continuation, client_call_details, request):
        return self._intercept(continuation, client_call_details, request, streaming=False)

    def intercept_unary_stream(self, continuation, client_call_details, request):
        return self._intercept(continuation, client_call_details, request, streaming=True)

    def _intercept(self, continuation, client_call_details, request, streaming: bool):
        try:
            from opentelemetry import trace
            from opentelemetry.propagate import get_global_textmap

            tracer = trace.get_tracer(self._service_name)
            method = client_call_details.method

            with tracer.start_as_current_span(method, kind=trace.SpanKind.CLIENT) as span:
                # Inject context into outgoing metadata
                carrier = _MetadataCarrier(client_call_details.metadata or [])
                get_global_textmap().inject(carrier)

                new_metadata = carrier.to_list()
                # Merge with existing metadata
                existing = list(client_call_details.metadata or [])
                new_metadata = existing + [
                    item for item in new_metadata
                    if item[0] not in {k for k, _ in existing}
                ]

                new_details = _ClientCallDetailsWithMetadata(client_call_details, new_metadata)

                try:
                    response = continuation(new_details, request)
                    return response
                except grpc.RpcError as exc:
                    span.record_exception(exc)
                    raise

        except ImportError:
            return continuation(client_call_details, request)


class _ClientCallDetailsWithMetadata(grpc.ClientCallDetails):
    def __init__(self, details: grpc.ClientCallDetails, metadata: list) -> None:
        self.method = details.method
        self.timeout = details.timeout
        self.metadata = metadata
        self.credentials = details.credentials
        self.wait_for_ready = details.wait_for_ready
        self.compression = details.compression


def make_tracing_client_interceptor(service_name: str) -> _TracingClientInterceptor:
    """
    Factory — returns a client interceptor to pass to grpc.intercept_channel().

    Example:
        channel = grpc.intercept_channel(
            grpc.insecure_channel("authz:50051"),
            make_tracing_client_interceptor("gateway"),
        )
    """
    return _TracingClientInterceptor(service_name)
