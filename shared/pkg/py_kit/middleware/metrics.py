"""
gRPC metrics interceptor — Python analogue of go-kit/middleware/metrics/metrics.go.

Wraps every unary RPC with:
  - request counter increment (before handler)
  - response counter increment with status label (after handler)
  - response time histogram observation (after handler)

Usage:
    from shared.pkg.py_kit.middleware.metrics import MetricsServerInterceptor
    from shared.pkg.py_kit import metric

    metric.init_otel_metrics(cfg)
    metric.init(cfg)

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        interceptors=[MetricsServerInterceptor()],
    )
"""

from __future__ import annotations

import time
import logging
from typing import Any, Callable

import grpc

from shared.pkg.py_kit import metric as _metric

log = logging.getLogger(__name__)


class MetricsServerInterceptor(grpc.ServerInterceptor):
    """
    gRPC server interceptor that records request/response metrics.
    Mirrors Go's metrics.MetricsInterceptor unary handler.
    """

    def intercept_service(
        self,
        continuation: Callable,
        handler_call_details: grpc.HandlerCallDetails,
    ):
        handler = continuation(handler_call_details)
        if handler is None:
            return handler

        method = handler_call_details.method

        def wrap(behavior: Callable):
            def new_behavior(request: Any, context: grpc.ServicerContext) -> Any:
                _metric.inc_request_counter()
                start = time.perf_counter()
                try:
                    result = behavior(request, context)
                    elapsed = time.perf_counter() - start
                    _metric.inc_response_counter("success", method)
                    _metric.histogram_response_time_observe("success", elapsed)
                    return result
                except Exception as exc:
                    elapsed = time.perf_counter() - start
                    _metric.inc_response_counter("error", method)
                    _metric.histogram_response_time_observe("error", elapsed)
                    raise

            return new_behavior

        if handler.unary_unary:
            return handler._replace(unary_unary=wrap(handler.unary_unary))
        return handler
