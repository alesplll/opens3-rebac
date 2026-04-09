"""
OTLP Metrics — Python analogue of go-kit/metric.

Exposes the same three instruments as the Go kit:
  - request_counter   (Int64Counter)
  - response_counter  (Int64Counter, labels: status, method)
  - response_time     (Float64Histogram, label: status)

Usage:
    from shared.pkg.py_kit.metric import init_otel_metrics, init
    from shared.pkg.py_kit.metric.config import MetricsConfig

    class MyCfg:
        def service_name(self):        return "authz"
        def service_version(self):     return "0.1.0"
        def otlp_endpoint(self):       return "otel-collector:4317"
        def service_environment(self): return "development"
        def push_interval_seconds(self): return 60.0

    provider = init_otel_metrics(MyCfg())   # sets up MeterProvider
    init(MyCfg())                           # creates named instruments

    # In request handler:
    from shared.pkg.py_kit import metric
    metric.inc_request_counter()
    metric.inc_response_counter("success", "/rebac.authz.v1.PermissionService/Check")
    metric.histogram_response_time_observe("success", 0.003)
"""

from __future__ import annotations

import logging
from typing import Optional

from .config import MetricsConfig

log = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Module-level singletons (mirrors Go package-level vars)
# ---------------------------------------------------------------------------

_meter = None
_request_counter = None
_response_counter = None
_histogram_response_time = None
_provider = None  # keep reference for shutdown

# Explicit bucket boundaries from Go kit
_HISTOGRAM_BOUNDARIES = [
    0.0001, 0.0002, 0.0004, 0.0008, 0.0016, 0.0032, 0.0064, 0.0128,
    0.0256, 0.0512, 0.1024, 0.2048, 0.4096, 0.8192, 1.6384, 3.2768,
]


def init_otel_metrics(cfg: MetricsConfig):
    """
    Initialise the global MeterProvider and wire it to the OTLP collector.
    Returns the MeterProvider (caller may want to shut it down on exit).
    Mirrors Go's metric.InitOTELMetrics().
    """
    global _meter, _provider

    try:
        from opentelemetry import metrics as otel_metrics
        from opentelemetry.sdk.metrics import MeterProvider
        from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
        from opentelemetry.sdk.metrics.view import View, ExplicitBucketHistogramAggregation
        from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
        from opentelemetry.sdk.resources import Resource, SERVICE_NAME, SERVICE_VERSION
        from opentelemetry.semconv.resource import ResourceAttributes
    except ImportError:
        log.warning(
            "opentelemetry packages not installed — metrics disabled. "
            "Install: opentelemetry-sdk opentelemetry-exporter-otlp-proto-grpc"
        )
        return None

    resource = Resource.create({
        SERVICE_NAME: cfg.service_name(),
        SERVICE_VERSION: cfg.service_version(),
        ResourceAttributes.DEPLOYMENT_ENVIRONMENT: cfg.service_environment(),
    })

    exporter = OTLPMetricExporter(
        endpoint=cfg.otlp_endpoint(),
        insecure=True,
    )

    reader = PeriodicExportingMetricReader(
        exporter,
        export_interval_millis=int(cfg.push_interval_seconds() * 1000),
    )

    # Apply explicit histogram buckets matching go-kit boundaries
    histogram_view = View(
        instrument_name=f"grpc_{cfg.service_name()}_histogram_response_time_seconds",
        aggregation=ExplicitBucketHistogramAggregation(boundaries=_HISTOGRAM_BOUNDARIES),
    )

    provider = MeterProvider(
        resource=resource,
        metric_readers=[reader],
        views=[histogram_view],
    )
    otel_metrics.set_meter_provider(provider)

    _meter = otel_metrics.get_meter(cfg.service_name())
    _provider = provider

    log.info("OpenTelemetry metrics initialised — endpoint=%s", cfg.otlp_endpoint())
    return provider


def init(cfg: MetricsConfig) -> None:
    """
    Create the standard gRPC instruments (counter, response counter, histogram).
    Must be called after init_otel_metrics(). Mirrors Go's metric.Init().
    """
    global _request_counter, _response_counter, _histogram_response_time

    meter = get_meter()
    if meter is None:
        return

    svc = cfg.service_name()

    _request_counter = meter.create_counter(
        name=f"grpc_{svc}_requests_total",
        description="Total number of incoming gRPC requests",
    )

    _response_counter = meter.create_counter(
        name=f"grpc_{svc}_response_total",
        description="Total number of gRPC responses by status and method",
    )

    _histogram_response_time = meter.create_histogram(
        name=f"grpc_{svc}_histogram_response_time_seconds",
        description="gRPC response latency distribution",
        unit="s",
    )


def get_meter():
    """Return the global Meter (noop if not initialised)."""
    if _meter is None:
        try:
            from opentelemetry import metrics as otel_metrics
            from opentelemetry.metrics import NoOpMeter  # noqa: F401
            return otel_metrics.get_meter("noop")
        except ImportError:
            return None
    return _meter


def inc_request_counter(attributes: Optional[dict] = None) -> None:
    """Increment total request counter. Mirrors Go's metric.IncRequestCounter()."""
    if _request_counter is None:
        return
    _request_counter.add(1, attributes=attributes or {})


def inc_response_counter(status: str, method: str) -> None:
    """
    Increment response counter with status (success|error) and gRPC method name.
    Mirrors Go's metric.IncResponseCounter().
    """
    if _response_counter is None:
        return
    _response_counter.add(1, attributes={"status": status, "method": method})


def histogram_response_time_observe(status: str, duration_seconds: float) -> None:
    """
    Record a latency observation.
    Mirrors Go's metric.HistogramResponseTimeObserve().
    """
    if _histogram_response_time is None:
        return
    _histogram_response_time.record(duration_seconds, attributes={"status": status})


def shutdown(timeout_seconds: float = 5.0) -> None:
    """Flush and shut down the MeterProvider. Call at service exit."""
    if _provider is None:
        return
    try:
        _provider.shutdown()
    except Exception as exc:
        log.error("Error shutting down MeterProvider: %s", exc)
