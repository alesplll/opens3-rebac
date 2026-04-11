"""ReBAC Auth Service entrypoint"""
import signal
import sys
from pathlib import Path
from concurrent import futures

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection

# Find repo root by locating the shared/ directory — works both locally and in Docker
for _p in Path(__file__).resolve().parents:
    if (_p / "shared").exists():
        REPO_ROOT = _p
        break
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from shared.pkg.py.authz.v1 import authz_pb2, authz_pb2_grpc
from shared.pkg.py_kit import logger, metric
from shared.pkg.py_kit.tracing import init_tracer, shutdown_tracer
from shared.pkg.py_kit.tracing.grpc_interceptor import TracingServerInterceptor
from shared.pkg.py_kit.middleware import MetricsServerInterceptor
from shared.pkg.py_kit.closer import configure_default, add_named
from internal import metric as authz_metrics
from internal.config import cfg
from internal.container import Container
from entrypoints.server.servicer import PermissionServiceServicer


def serve():
    # ── Observability init (порядок как в Go: logger → metrics → tracing) ──

    logger.init(cfg)

    provider = metric.init_otel_metrics(cfg)
    metric.init(cfg)
    authz_metrics.init()

    init_tracer(cfg)

    # ── Graceful shutdown ──────────────────────────────────────────────────

    closer = configure_default(signal.SIGINT, signal.SIGTERM)

    add_named("otel-tracing", shutdown_tracer)
    add_named("otel-metrics", lambda: metric.shutdown())

    # ── Dependencies ──────────────────────────────────────────────────────

    container = Container(cfg)
    add_named("neo4j", container.neo4j_store.close)

    # ── gRPC server ────────────────────────────────────────────────────────

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        interceptors=[
            TracingServerInterceptor(cfg.service_name()),
            MetricsServerInterceptor(),
        ],
    )

    add_named("grpc-server", lambda: server.stop(grace=5).wait())

    authz_pb2_grpc.add_PermissionServiceServicer_to_server(
        PermissionServiceServicer(container), server
    )

    # gRPC Health
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set(
        authz_pb2.DESCRIPTOR.services_by_name["PermissionService"].full_name,
        health_pb2.HealthCheckResponse.SERVING,
    )

    # gRPC Reflection
    SERVICE_NAMES = (
        authz_pb2.DESCRIPTOR.services_by_name["PermissionService"].full_name,
        health_pb2.DESCRIPTOR.services_by_name["Health"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    server.add_insecure_port(f"[::]:{cfg.grpc_port()}")
    server.start()

    logger.info({}, "ReBAC Auth Service started", port=cfg.grpc_port())

    try:
        server.wait_for_termination()
    finally:
        logger.info({}, "Shutting down...")
        logger.sync()


def run_server():
    """Entry point for rebac-server console script."""
    serve()


if __name__ == "__main__":
    serve()
