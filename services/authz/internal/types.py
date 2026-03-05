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

    def __str__(self) -> str:
        level_str = f" level={self.level}" if self.level else ""
        return f"({self.subject} {self.relation}→ {self.object}{level_str})"

