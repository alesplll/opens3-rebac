"""Unit tests for PermissionService with mocked store and cache."""
from unittest.mock import MagicMock, patch

import pytest

from internal.types import Tuple
from internal.rebac.model import PermissionService
from internal.gen import authz_pb2


class TestPermissionServiceCheck:
    def test_no_store_denies(self):
        svc = PermissionService(store=None, cache=None, audit_producer=None)
        assert svc.check("user:alice", "read", "doc:1") is False

    def test_delegates_to_store_and_caches_result(self):
        store = MagicMock()
        store.check.return_value = True
        cache = MagicMock()
        cache.get.return_value = None

        svc = PermissionService(store=store, cache=cache, audit_producer=None)
        result = svc.check("user:alice", "read", "doc:1")

        assert result is True
        store.check.assert_called_once_with("user:alice", "read", "doc:1")
        cache.get.assert_called_once_with("user:alice", "read", "doc:1")
        cache.set.assert_called_once_with("user:alice", "read", "doc:1", True, ttl_seconds=30)

    def test_check_sends_audit_event_on_allow(self):
        store = MagicMock()
        store.check.return_value = True
        cache = MagicMock()
        cache.get.return_value = None
        audit = MagicMock()

        svc = PermissionService(store=store, cache=cache, audit_producer=audit)
        result = svc.check("user:alice", "read", "doc:1")

        assert result is True
        audit.send_decision_event.assert_called_once_with("user:alice", "read", "doc:1", True)

    def test_check_sends_audit_event_on_deny(self):
        store = MagicMock()
        store.check.return_value = False
        cache = MagicMock()
        cache.get.return_value = None
        audit = MagicMock()

        svc = PermissionService(store=store, cache=cache, audit_producer=audit)
        result = svc.check("user:bob", "write", "doc:2")

        assert result is False
        audit.send_decision_event.assert_called_once_with("user:bob", "write", "doc:2", False)

    def test_check_sends_audit_event_on_cache_hit(self):
        store = MagicMock()
        cache = MagicMock()
        cache.get.return_value = True  # cache hit
        audit = MagicMock()

        svc = PermissionService(store=store, cache=cache, audit_producer=audit)
        result = svc.check("user:alice", "read", "doc:1")

        assert result is True
        store.check.assert_not_called()
        audit.send_decision_event.assert_called_once_with("user:alice", "read", "doc:1", True)

    def test_returns_cached_result_without_calling_store(self):
        store = MagicMock()
        cache = MagicMock()
        cache.get.return_value = False

        svc = PermissionService(store=store, cache=cache, audit_producer=None)
        result = svc.check("user:bob", "write", "doc:2")

        assert result is False
        store.check.assert_not_called()
        cache.set.assert_not_called()


class TestPermissionServiceWriteTuple:
    def test_no_store_raises(self):
        svc = PermissionService(store=None, cache=None, audit_producer=None)
        with pytest.raises(RuntimeError, match="No storage"):
            svc.write_tuple(Tuple("user:a", "MEMBER_OF", "group:g", level=None))

    def test_delegates_to_store_and_emits_audit(self):
        store = MagicMock()
        store.write_tuple.return_value = True
        audit = MagicMock()

        svc = PermissionService(store=store, cache=MagicMock(), audit_producer=audit)
        t = Tuple("group:devops", "HAS_PERMISSION", "resource:server-1", level="admin")
        result = svc.write_tuple(t)

        assert result is True
        store.write_tuple.assert_called_once_with(t)
        audit.send_tuple_event.assert_called_once_with(t, "tuple_written")


class TestPermissionServiceDeleteTuple:
    def test_no_store_raises(self):
        svc = PermissionService(store=None, cache=None, audit_producer=None)
        with pytest.raises(RuntimeError, match="No storage"):
            svc.delete_tuple(Tuple("user:a", "MEMBER_OF", "group:g"))

    def test_delegates_to_store_and_emits_audit(self):
        store = MagicMock()
        store.delete_tuple.return_value = True
        audit = MagicMock()

        svc = PermissionService(store=store, cache=None, audit_producer=audit)
        t = Tuple("user:alice", "MEMBER_OF", "group:dev")
        result = svc.delete_tuple(t)

        assert result is True
        store.delete_tuple.assert_called_once_with(t)
        audit.send_tuple_event.assert_called_once_with(t, "tuple_removed")

    def test_no_audit_if_delete_fails(self):
        store = MagicMock()
        store.delete_tuple.return_value = False
        audit = MagicMock()

        svc = PermissionService(store=store, cache=None, audit_producer=audit)
        result = svc.delete_tuple(Tuple("user:alice", "MEMBER_OF", "group:dev"))

        assert result is False
        audit.send_tuple_event.assert_not_called()


class TestPermissionServiceReadTuples:
    def test_no_store_returns_empty(self):
        svc = PermissionService(store=None, cache=None, audit_producer=None)
        assert svc.read_tuples("user:alice") == []

    def test_delegates_to_store(self):
        store = MagicMock()
        store.read_tuples.return_value = [
            Tuple("user:alice", "MEMBER_OF", "group:dev", level=None),
        ]

        svc = PermissionService(store=store, cache=None, audit_producer=None)
        result = svc.read_tuples("user:alice")

        assert len(result) == 1
        assert result[0].subject == "user:alice" and result[0].relation == "MEMBER_OF"
        store.read_tuples.assert_called_once_with("user:alice")


class TestPermissionServiceServicerHealthCheck:
    """Unit tests for HealthCheck RPC handler in PermissionServiceServicer."""

    def _make_servicer(self, neo4j_mock, redis_mock):
        """Create servicer with mocked dependencies."""
        from entrypoints.server.main import PermissionServiceServicer
        with patch("entrypoints.server.main.Neo4jStore", return_value=neo4j_mock), \
             patch("entrypoints.server.main.RedisDecisionCache", return_value=redis_mock), \
             patch("entrypoints.server.main.AuditProducer"):
            return PermissionServiceServicer()

    def test_returns_serving_when_all_healthy(self):
        neo4j = MagicMock()
        redis = MagicMock()

        servicer = self._make_servicer(neo4j, redis)
        response = servicer.HealthCheck(MagicMock(), MagicMock())

        assert response.status == authz_pb2.HealthCheckResponse.SERVING

    def test_returns_not_serving_when_neo4j_down(self):
        neo4j = MagicMock()
        neo4j.driver.verify_connectivity.side_effect = Exception("Neo4j unreachable")
        redis = MagicMock()

        servicer = self._make_servicer(neo4j, redis)
        response = servicer.HealthCheck(MagicMock(), MagicMock())

        assert response.status == authz_pb2.HealthCheckResponse.NOT_SERVING

    def test_returns_not_serving_when_redis_down(self):
        neo4j = MagicMock()
        redis = MagicMock()
        redis._client.ping.side_effect = Exception("Redis unreachable")

        servicer = self._make_servicer(neo4j, redis)
        response = servicer.HealthCheck(MagicMock(), MagicMock())

        assert response.status == authz_pb2.HealthCheckResponse.NOT_SERVING

    def test_returns_not_serving_when_both_down(self):
        neo4j = MagicMock()
        neo4j.driver.verify_connectivity.side_effect = Exception("Neo4j unreachable")
        redis = MagicMock()
        redis._client.ping.side_effect = Exception("Redis unreachable")

        servicer = self._make_servicer(neo4j, redis)
        response = servicer.HealthCheck(MagicMock(), MagicMock())

        assert response.status == authz_pb2.HealthCheckResponse.NOT_SERVING
