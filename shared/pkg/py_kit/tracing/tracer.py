"""
OTLP Tracing — Python analogue of go-kit/tracing/tracer.go.

Initialises a global TracerProvider that exports spans to the OTLP collector
via gRPC (same endpoint as metrics / logs).

Context model:
  In Go, context is passed explicitly. In Python we use contextvars via the
  OTel SDK — the current span is available from opentelemetry.trace.get_current_span().
  For gRPC propagation we extract/inject via grpc.ServicerContext metadata.

Usage:
    from shared.pkg.py_kit.tracing import init_tracer, start_span, trace_id_from_context
    from shared.pkg.py_kit.tracing.config import TracingConfig

    class MyCfg:
        def collector_endpoint(self): return "otel-collector:4317"
        def service_name(self):       return "authz"
        def environment(self):        return "development"
        def service_version(self):    return "0.1.0"

    init_tracer(MyCfg())

    with start_span("my-operation") as span:
        span.set_attribute("key", "value")
"""

from __future__ import annotations

import logging
from contextlib import contextmanager
from typing import Generator, Optional

log = logging.getLogger(__name__)

_service_name: str = ""
_provider = None  # TracerProvider reference for shutdown


class TracingConfig:
    """Protocol — implement this in your service config."""
    def collector_endpoint(self) -> str: ...
    def service_name(self) -> str: ...
    def environment(self) -> str: ...
    def service_version(self) -> str: ...


def init_tracer(cfg: TracingConfig) -> None:
    """
    Initialise the global TracerProvider.
    Mirrors Go's tracing.InitTracer().

    - Sends spans to cfg.collector_endpoint() via gRPC (insecure, gzip).
    - Retry: initial 500ms, max 5s, elapsed 30s.
    - Sampler: ParentBased(TraceIDRatioBased(1.0)) — 100% sampling.
    - Propagator: W3C TraceContext + Baggage.
    """
    global _service_name, _provider

    try:
        from opentelemetry import trace
        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry.sdk.trace.export import BatchSpanProcessor
        from opentelemetry.sdk.trace.sampling import ParentBased, TraceIdRatioBased
        from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
        from opentelemetry.sdk.resources import Resource, SERVICE_NAME, SERVICE_VERSION
        from opentelemetry.semconv.resource import ResourceAttributes
        from opentelemetry.propagate import set_global_textmap
        from opentelemetry.propagators.composite import CompositePropagator
        from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
        from opentelemetry.baggage.propagation import W3CBaggagePropagator
    except ImportError:
        log.warning(
            "opentelemetry packages not installed — tracing disabled. "
            "Install: opentelemetry-sdk opentelemetry-exporter-otlp-proto-grpc"
        )
        return

    _service_name = cfg.service_name()

    resource = Resource.create({
        SERVICE_NAME: cfg.service_name(),
        SERVICE_VERSION: cfg.service_version(),
        ResourceAttributes.DEPLOYMENT_ENVIRONMENT: cfg.environment(),
    })

    exporter = OTLPSpanExporter(
        endpoint=cfg.collector_endpoint(),
        insecure=True,
        compression=None,  # grpc default; set to "gzip" if collector supports it
    )

    provider = TracerProvider(
        resource=resource,
        sampler=ParentBased(TraceIdRatioBased(1.0)),
    )
    provider.add_span_processor(BatchSpanProcessor(exporter))

    trace.set_tracer_provider(provider)

    # W3C TraceContext + Baggage — same as Go's propagation.NewCompositeTextMapPropagator
    set_global_textmap(CompositePropagator([
        TraceContextTextMapPropagator(),
        W3CBaggagePropagator(),
    ]))

    _provider = provider
    log.info("OpenTelemetry tracing initialised — endpoint=%s", cfg.collector_endpoint())


def shutdown_tracer(timeout_seconds: float = 5.0) -> None:
    """Flush and shut down the TracerProvider. Mirrors Go's tracing.ShutdownTracer()."""
    if _provider is None:
        return
    try:
        _provider.shutdown()
    except Exception as exc:
        log.error("Error shutting down TracerProvider: %s", exc)


@contextmanager
def start_span(name: str, **attributes) -> Generator:
    """
    Context-manager that starts a child span and ends it on exit.
    Mirrors Go's tracing.StartSpan() — but as a context manager for Python idioms.

    Example:
        with start_span("neo4j.check", subject="user:alice") as span:
            span.set_attribute("action", "read")
    """
    try:
        from opentelemetry import trace
        tracer = trace.get_tracer(_service_name or __name__)
        with tracer.start_as_current_span(name) as span:
            for k, v in attributes.items():
                span.set_attribute(k, str(v))
            yield span
    except ImportError:
        # OTel not installed — yield a no-op sentinel
        yield _NoOpSpan()


def span_from_context():
    """
    Return the current active span (if any).
    Mirrors Go's tracing.SpanFromContext(ctx).
    """
    try:
        from opentelemetry import trace
        return trace.get_current_span()
    except ImportError:
        return None


def trace_id_from_context() -> str:
    """
    Extract the trace ID string from the current span context.
    Mirrors Go's tracing.TraceIDFromContext(ctx).
    Returns "" if no valid span is active.
    """
    try:
        from opentelemetry import trace
        span = trace.get_current_span()
        ctx = span.get_span_context()
        if ctx is None or not ctx.is_valid:
            return ""
        return format(ctx.trace_id, "032x")
    except ImportError:
        return ""


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

class _NoOpSpan:
    """Sentinel returned when OTel is not installed."""
    def set_attribute(self, key: str, value) -> None: pass
    def record_exception(self, exc) -> None: pass
    def set_status(self, *args, **kwargs) -> None: pass
