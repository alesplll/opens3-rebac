"""Shared ReBAC types"""
from dataclasses import dataclass
from typing import List, Dict, Optional
import logging

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class Tuple:
    """Atomic relationship tuple: (subject, relation, object) with optional level for HAS_PERMISSION."""
    subject: str
    relation: str
    object: str
    level: Optional[str] = None  # For HAS_PERMISSION: "read" | "write" | "create" | "delete" | "admin"
    # Who performed the change, taken from the caller's access token.
    # None means unknown, which is a distinct state from "nobody".
    actor: Optional[str] = None

    def __str__(self) -> str:
        level_str = f" level={self.level}" if self.level else ""
        actor_str = f" by={self.actor}" if self.actor else ""
        return f"({self.subject} {self.relation}→ {self.object}{level_str}{actor_str})"

