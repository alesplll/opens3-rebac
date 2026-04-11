"""ReBAC core model and service"""
import time
from typing import List, Optional

from internal.rebac.interfaces import GraphStore
from internal.types import Tuple
from internal.cache.interfaces import DecisionCache
from internal.kafka.producer import AuditProducer
from internal import metric as authz_metrics

from shared.pkg.py_kit import logger
from shared.pkg.py_kit.tracing import start_span


class PermissionService:
    """ReBAC authorization service"""

    def __init__(
        self,
        store: Optional[GraphStore] = None,
        cache: Optional[DecisionCache] = None,
        audit_producer: Optional[AuditProducer] = None,
    ):
        self._store = store
        self._cache = cache
        self._audit_producer = audit_producer

    def write_tuple(self, tuple_: Tuple) -> bool:
        """Write relationship tuple to Neo4j."""
        logger.info({}, "Writing tuple", subject=tuple_.subject, relation=tuple_.relation, object=tuple_.object)

        if not self._store:
            raise RuntimeError("No storage configured")

        with start_span("rebac.write_tuple", subject=tuple_.subject, relation=tuple_.relation, object=tuple_.object):
            t0 = time.perf_counter()
            success = self._store.write_tuple(tuple_)
            authz_metrics.record_neo4j_query("write", time.perf_counter() - t0)

            if success and self._audit_producer:
                with start_span("audit.emit", event="tuple_written"):
                    self._audit_producer.send_tuple_event(tuple_, "tuple_written")

        logger.debug({}, "Write tuple result", success=success)
        return success

    def delete_tuple(self, tuple_: Tuple) -> bool:
        """Delete relationship tuple from Neo4j."""
        logger.info({}, "Deleting tuple", subject=tuple_.subject, relation=tuple_.relation, object=tuple_.object)

        if not self._store:
            raise RuntimeError("No storage configured")

        with start_span("rebac.delete_tuple", subject=tuple_.subject, relation=tuple_.relation, object=tuple_.object):
            t0 = time.perf_counter()
            success = self._store.delete_tuple(tuple_)
            authz_metrics.record_neo4j_query("delete", time.perf_counter() - t0)

            if success and self._audit_producer:
                with start_span("audit.emit", event="tuple_removed"):
                    self._audit_producer.send_tuple_event(tuple_, "tuple_removed")

        logger.debug({}, "Delete tuple result", success=success)
        return success

    def read_tuples(self, subject: str) -> List[Tuple]:
        """Read all outgoing relationships for subject."""
        logger.debug({}, "Reading tuples", subject=subject)

        if not self._store:
            return []

        with start_span("rebac.read_tuples", subject=subject):
            t0 = time.perf_counter()
            tuples = self._store.read_tuples(subject)
            authz_metrics.record_neo4j_query("read", time.perf_counter() - t0)

        logger.debug({}, "Read tuples result", subject=subject, count=len(tuples))
        return tuples

    def check(self, subject: str, action: str, object: str) -> bool:
        """
        Check if subject can perform action on object.
        Flow: Redis cache → Neo4j graph → cache write → audit emit.
        """
        if not self._store:
            logger.warn({}, "No storage configured — denying access", subject=subject, action=action, object=object)
            return False

        with start_span("rebac.check", subject=subject, action=action, object=object) as root_span:

            # ── 1. Cache lookup ───────────────────────────────────────────
            allowed: Optional[bool] = None
            cache_hit = False

            if self._cache:
                with start_span("cache.lookup", subject=subject, action=action, object=object) as cache_span:
                    cached = self._cache.get(subject, action, object)
                    if cached is not None:
                        allowed = cached
                        cache_hit = True
                        cache_span.set_attribute("result", "hit")
                        authz_metrics.record_cache_hit()
                        logger.debug(
                            {}, "Cache hit",
                            subject=subject, action=action, object=object,
                            decision="allow" if cached else "deny",
                        )
                    else:
                        cache_span.set_attribute("result", "miss")
                        authz_metrics.record_cache_miss()

            # ── 2. Neo4j check on cache miss ──────────────────────────────
            if allowed is None:
                with start_span("neo4j.check", subject=subject, action=action, object=object) as neo_span:
                    t0 = time.perf_counter()
                    allowed = self._store.check(subject, action, object)
                    elapsed = time.perf_counter() - t0
                    authz_metrics.record_neo4j_query("check", elapsed)
                    neo_span.set_attribute("result", "allow" if allowed else "deny")
                    neo_span.set_attribute("duration_ms", round(elapsed * 1000, 2))

                if self._cache:
                    self._cache.set(subject, action, object, allowed, ttl_seconds=30)

            # ── 3. Record decision metric ─────────────────────────────────
            result_str = "allow" if allowed else "deny"
            authz_metrics.record_decision(action, result_str)
            root_span.set_attribute("result", result_str)
            root_span.set_attribute("cache_hit", cache_hit)

            logger.info(
                {}, "Authorization decision",
                subject=subject, action=action, object=object,
                result=result_str, cache_hit=cache_hit,
            )

            # ── 4. Audit emit ─────────────────────────────────────────────
            if self._audit_producer:
                with start_span("audit.emit", result=result_str):
                    self._audit_producer.send_decision_event(subject, action, object, allowed)

        return allowed

    def close(self) -> None:
        """Close underlying storage."""
        if self._store:
            self._store.close()
