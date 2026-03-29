"""ReBAC Auth Service entrypoint"""
import logging
import os
import sys
from pathlib import Path
from concurrent import futures

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection

REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from shared.pkg.py.authz.v1 import authz_pb2, authz_pb2_grpc
from internal.rebac.model import PermissionService
from internal.neo4j.store import Neo4jStore
from internal.types import Tuple
from internal.cache.redis_cache import RedisDecisionCache
from internal.kafka.producer import AuditProducer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

NEO4J_URI = os.environ.get("NEO4J_URI", "bolt://localhost:7687")
NEO4J_USER = os.environ.get("NEO4J_USER", "neo4j")
NEO4J_PASSWORD = os.environ.get("NEO4J_PASSWORD", "password123")
REDIS_HOST = os.environ.get("REDIS_HOST", "localhost")
REDIS_PORT = int(os.environ.get("REDIS_PORT", "6379"))
KAFKA_BOOTSTRAP = os.environ.get("KAFKA_BOOTSTRAP", "localhost:9092")
GRPC_PORT = os.environ.get("GRPC_PORT", "50051")


class PermissionServiceServicer(authz_pb2_grpc.PermissionServiceServicer):
    """gRPC service implementation"""

    def __init__(self):
        self._neo4j_store = Neo4jStore(uri=NEO4J_URI, user=NEO4J_USER, password=NEO4J_PASSWORD)
        self._cache = RedisDecisionCache(host=REDIS_HOST, port=REDIS_PORT)
        audit_producer = AuditProducer(KAFKA_BOOTSTRAP)
        self.rebac = PermissionService(
            store=self._neo4j_store, cache=self._cache, audit_producer=audit_producer
        )

    def Check(self, request, context):
        allowed = self.rebac.check(request.subject, request.action, request.object)
        return authz_pb2.CheckResponse(
            allowed=allowed, reason="Neo4j ReBAC (transitive + HAS_PERMISSION)"
        )

    def WriteTuple(self, request, context):
        level = request.level if request.level else None
        tuple_ = Tuple(request.subject, request.relation, request.object, level=level)
        success = self.rebac.write_tuple(tuple_)
        return authz_pb2.WriteTupleResponse(success=success)

    def DeleteTuple(self, request, context):
        tuple_ = Tuple(request.subject, request.relation, request.object)
        success = self.rebac.delete_tuple(tuple_)
        return authz_pb2.DeleteTupleResponse(success=success)

    def Read(self, request, context):
        tuples = self.rebac.read_tuples(request.subject)
        response = authz_pb2.ReadResponse()
        for t in tuples:
            rt = response.tuples.add(subject=t.subject, relation=t.relation, object=t.object)
            if t.level:
                rt.level = t.level
        return response

    def HealthCheck(self, request, context):
        status = authz_pb2.HealthCheckResponse.SERVING
        try:
            self._neo4j_store.driver.verify_connectivity()
        except Exception as e:
            logger.warning("Neo4j health check failed: %s", e)
            status = authz_pb2.HealthCheckResponse.NOT_SERVING
        try:
            self._cache._client.ping()
        except Exception as e:
            logger.warning("Redis health check failed: %s", e)
            status = authz_pb2.HealthCheckResponse.NOT_SERVING
        return authz_pb2.HealthCheckResponse(status=status)


def serve():
    """Start gRPC server"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    # Business service
    authz_pb2_grpc.add_PermissionServiceServicer_to_server(
        PermissionServiceServicer(), server
    )

    # gRPC Health (standard protocol — used by K8s liveness/readiness probes)
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set(
        authz_pb2.DESCRIPTOR.services_by_name["PermissionService"].full_name,
        health_pb2.HealthCheckResponse.SERVING,
    )

    # gRPC Reflection (grpcurl / Postman)
    SERVICE_NAMES = (
        authz_pb2.DESCRIPTOR.services_by_name["PermissionService"].full_name,
        health_pb2.DESCRIPTOR.services_by_name["Health"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    server.add_insecure_port(f"[::]:{GRPC_PORT}")
    server.start()
    logger.info("ReBAC Auth Service gRPC :%s", GRPC_PORT)
    try:
        server.wait_for_termination()
    finally:
        logger.info("Shutting down...")


def run_server():
    """Entry point for rebac-server console script."""
    serve()


if __name__ == "__main__":
    serve()
