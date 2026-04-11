"""gRPC handler — translates protobuf requests into PermissionService calls."""
from shared.pkg.py.authz.v1 import authz_pb2, authz_pb2_grpc
from shared.pkg.py_kit import logger
from internal.container import Container
from internal.types import Tuple


class PermissionServiceServicer(authz_pb2_grpc.PermissionServiceServicer):

    def __init__(self, container: Container):
        self._rebac = container.rebac
        self._neo4j_store = container.neo4j_store
        self._cache = container.cache

    def Check(self, request, context):
        logger.debug({}, "Check RPC", subject=request.subject, action=request.action, object=request.object)
        allowed = self._rebac.check(request.subject, request.action, request.object)
        logger.info({}, "Check RPC done", subject=request.subject, action=request.action, object=request.object, allowed=allowed)
        return authz_pb2.CheckResponse(
            allowed=allowed, reason="Neo4j ReBAC (transitive + HAS_PERMISSION)"
        )

    def WriteTuple(self, request, context):
        logger.debug({}, "WriteTuple RPC", subject=request.subject, relation=request.relation, object=request.object)
        level = request.level if request.level else None
        tuple_ = Tuple(request.subject, request.relation, request.object, level=level)
        success = self._rebac.write_tuple(tuple_)
        logger.info({}, "WriteTuple RPC done", success=success)
        return authz_pb2.WriteTupleResponse(success=success)

    def DeleteTuple(self, request, context):
        logger.debug({}, "DeleteTuple RPC", subject=request.subject, relation=request.relation, object=request.object)
        tuple_ = Tuple(request.subject, request.relation, request.object)
        success = self._rebac.delete_tuple(tuple_)
        logger.info({}, "DeleteTuple RPC done", success=success)
        return authz_pb2.DeleteTupleResponse(success=success)

    def Read(self, request, context):
        logger.debug({}, "Read RPC", subject=request.subject)
        tuples = self._rebac.read_tuples(request.subject)
        response = authz_pb2.ReadResponse()
        for t in tuples:
            rt = response.tuples.add(subject=t.subject, relation=t.relation, object=t.object)
            if t.level:
                rt.level = t.level
        logger.debug({}, "Read RPC done", count=len(tuples))
        return response

    def HealthCheck(self, request, context):
        status = authz_pb2.HealthCheckResponse.SERVING
        try:
            self._neo4j_store.driver.verify_connectivity()
        except Exception as e:
            logger.warn({}, "Neo4j health check failed", error=str(e))
            status = authz_pb2.HealthCheckResponse.NOT_SERVING
        try:
            self._cache._client.ping()
        except Exception as e:
            logger.warn({}, "Redis health check failed", error=str(e))
            status = authz_pb2.HealthCheckResponse.NOT_SERVING
        return authz_pb2.HealthCheckResponse(status=status)
