"""ReBAC core model and service"""
from dataclasses import dataclass
from typing import List, Optional
import logging
from internal.rebac.interfaces import GraphStore
from internal.types import Tuple
from internal.cache.interfaces import DecisionCache
from internal.kafka.producer import AuditProducer

logger = logging.getLogger(__name__)

class PermissionService:
    """ReBAC authorization service"""
    
    def __init__(self, store: Optional[GraphStore] = None, 
                 cache: Optional[DecisionCache] = None,
                 audit_producer: Optional[AuditProducer] = None):
        """
        Initialize with graph store and optional decision cache.

        Args:
            store: GraphStore implementation (Neo4jStore, etc.).
            cache: DecisionCache implementation (RedisDecisionCache, etc.).
        """
        self._store = store
        self._cache = cache
        self._audit_producer = audit_producer

    def write_tuple(self, tuple_: Tuple) -> bool:
        """Write relationship tuple"""
        if not self._store:
            raise RuntimeError("No storage configured")
        logger.info(f"Write tuple: {tuple_}")
        success = self._store.write_tuple(tuple_)

        if self._audit_producer and success:
            self._audit_producer.send_tuple_event(tuple_, "tuple_written")

        return success

    def delete_tuple(self, tuple_: Tuple) -> bool:
        """Delete relationship tuple"""
        if not self._store:
            raise RuntimeError("No storage configured")
        logger.info(f"Delete tuple: {tuple_}")
        success = self._store.delete_tuple(tuple_)

        if self._audit_producer and success:
            self._audit_producer.send_tuple_event(tuple_, "tuple_removed")

        return success

    def read_tuples(self, subject: str) -> List[Tuple]:
        """Read all outgoing relationships for subject"""
        if not self._store:
            return []
        logger.debug(f"Read tuples for: {subject}")
        return self._store.read_tuples(subject)

    def check(self, subject: str, action: str, object: str) -> bool:
        """Check if subject can perform action on object with caching and audit."""
        if not self._store:
            logger.warning("No storage - denying access")
            return False

        # 1) cache lookup
        allowed = None
        if self._cache:
            cached = self._cache.get(subject, action, object)
            if cached is not None:
                logger.info(
                    "Authorization (cached): %s %s %s -> %s",
                    subject, action, object, "ALLOW" if cached else "DENY",
                )
                allowed = cached

        # 2) storage check on cache miss
        if allowed is None:
            logger.info("Authorization (store): %s %s %s", subject, action, object)
            allowed = self._store.check(subject, action, object)
            if self._cache:
                self._cache.set(subject, action, object, allowed, ttl_seconds=30)

        # 3) audit every decision
        if self._audit_producer:
            self._audit_producer.send_decision_event(subject, action, object, allowed)

        logger.info("Authorization result: %s", "ALLOW" if allowed else "DENY")
        return allowed

    def close(self) -> None:
        """Close underlying storage"""
        if self._store:
            self._store.close()

