from typing import Optional
import logging
import redis
from internal.cache.interfaces import DecisionCache

logger = logging.getLogger(__name__)

class RedisDecisionCache(DecisionCache):
    """Decision cache backed by Redis."""

    def __init__(self, host: str = "localhost", port: int = 6379, db: int = 0, prefix: str = "auth_decision"):
        """
        Initialize Redis client.

        Args:
            host: Redis host.
            port: Redis port.
            db: Redis logical database index.
            prefix: Key prefix for namespacing.
        """
        self._client = redis.Redis(host=host, port=port, db=db, decode_responses=True)
        self._prefix = prefix
        logger.info(f"Connected to Redis: {host}:{port}, db {db}")


    def _make_key(self, subject: str, action: str, object_: str) -> str:
        """Build cache key for decision."""
        return f"{self._prefix}:{subject}:{action}:{object_}"

    def get(self, subject: str, action: str, object_: str) -> Optional[bool]:
        """Return cached decision or None if missing."""
        key = self._make_key(subject, action, object_)
        value = self._client.get(key)
        if value is None:
            logger.debug(f"Decision cache MISS for key={key}")
            return None
        logger.debug(f"Decision cache HIT for key={key}: {value}")
        return value == "1"

    def set(self, subject: str, action: str, object_: str, allowed: bool, ttl_seconds: int) -> None:
        """Store decision with TTL."""
        key = self._make_key(subject, action, object_)
        value = "1" if allowed else "0"
        self._client.set(name=key, value=value, ex=ttl_seconds)
        logger.debug(f"Decision cached key={key} value={value} ttl={ttl_seconds}s")

