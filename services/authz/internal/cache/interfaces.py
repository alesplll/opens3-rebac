from abc import ABC, abstractmethod
from typing import Optional

class DecisionCache(ABC):
    """Contract for decision cache implementations."""

    @abstractmethod
    def get(self, subject: str, action: str, object_: str) -> Optional[bool]:
        """Return cached decision or None if not present."""
        raise NotImplementedError

    @abstractmethod
    def set(self, subject: str, action: str, object_: str, allowed: bool, ttl_seconds: int) -> None:
        """Store decision with expiration."""
        raise NotImplementedError

