"""gRPC handler — translates protobuf requests into PermissionService calls."""
import grpc
from shared.pkg.py.authz.v1 import authz_pb2, authz_pb2_grpc
from shared.pkg.py_kit import logger
from internal.container import Container
from internal.types import Tuple

_ACTION_TO_STR = {
    authz_pb2.Action.ACTION_READ:   "read",
    authz_pb2.Action.ACTION_WRITE:  "write",
    authz_pb2.Action.ACTION_CREATE: "create",
    authz_pb2.Action.ACTION_DELETE: "delete",
    authz_pb2.Action.ACTION_ADMIN:  "admin",
}

_RELATION_TO_STR = {
    authz_pb2.Relation.RELATION_MEMBER_OF:      "MEMBER_OF",
    authz_pb2.Relation.RELATION_HAS_PERMISSION: "HAS_PERMISSION",
    authz_pb2.Relation.RELATION_PARENT_OF:      "PARENT_OF",
}

_STR_TO_RELATION = {v: k for k, v in _RELATION_TO_STR.items()}

_LEVEL_TO_STR = {
    authz_pb2.PermissionLevel.PERMISSION_LEVEL_READ:   "read",
    authz_pb2.PermissionLevel.PERMISSION_LEVEL_WRITE:  "write",
    authz_pb2.PermissionLevel.PERMISSION_LEVEL_CREATE: "create",
    authz_pb2.PermissionLevel.PERMISSION_LEVEL_DELETE: "delete",
    authz_pb2.PermissionLevel.PERMISSION_LEVEL_ADMIN:  "admin",
}

_STR_TO_LEVEL = {v: k for k, v in _LEVEL_TO_STR.items()}


class PermissionServiceServicer(authz_pb2_grpc.PermissionServiceServicer):

    def __init__(self, container: Container):
        self._rebac = container.rebac

    def Check(self, request, context):
        action = _ACTION_TO_STR.get(request.action)
        if not action:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "action is required")

        logger.debug({}, "Check RPC", subject=request.subject, action=action, object=request.object)
        allowed, reason = self._rebac.check(request.subject, action, request.object)
        logger.info({}, "Check RPC done", subject=request.subject, action=action, object=request.object, allowed=allowed)
        return authz_pb2.CheckResponse(allowed=allowed, reason=reason)

    def WriteTuple(self, request, context):
        relation = _RELATION_TO_STR.get(request.relation)
        if not relation:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "relation is required")

        level = _LEVEL_TO_STR.get(request.level) if request.level else None
        if relation == "HAS_PERMISSION" and not level:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "level is required for HAS_PERMISSION")

        logger.debug({}, "WriteTuple RPC", subject=request.subject, relation=relation, object=request.object)
        tuple_ = Tuple(request.subject, relation, request.object, level=level)
        success = self._rebac.write_tuple(tuple_)
        logger.info({}, "WriteTuple RPC done", success=success)
        return authz_pb2.WriteTupleResponse(success=success)

    def DeleteTuple(self, request, context):
        relation = _RELATION_TO_STR.get(request.relation)
        if not relation:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "relation is required")

        logger.debug({}, "DeleteTuple RPC", subject=request.subject, relation=relation, object=request.object)
        tuple_ = Tuple(request.subject, relation, request.object)
        success = self._rebac.delete_tuple(tuple_)
        logger.info({}, "DeleteTuple RPC done", success=success)
        return authz_pb2.DeleteTupleResponse(success=success)

    def Read(self, request, context):
        logger.debug({}, "Read RPC", subject=request.subject)
        tuples = self._rebac.read_tuples(request.subject)
        response = authz_pb2.ReadResponse()
        for t in tuples:
            rt = response.tuples.add(
                subject=t.subject,
                relation=_STR_TO_RELATION.get(t.relation, authz_pb2.Relation.RELATION_UNSPECIFIED),
                object=t.object,
            )
            if t.level:
                rt.level = _STR_TO_LEVEL.get(t.level, authz_pb2.PermissionLevel.PERMISSION_LEVEL_UNSPECIFIED)
        logger.debug({}, "Read RPC done", count=len(tuples))
        return response

    def HealthCheck(self, request, context):
        try:
            self._rebac.health_check()
            return authz_pb2.HealthCheckResponse(status=authz_pb2.HealthCheckResponse.SERVING)
        except Exception as e:
            logger.warn({}, "Health check failed", error=str(e))
            return authz_pb2.HealthCheckResponse(status=authz_pb2.HealthCheckResponse.NOT_SERVING)
