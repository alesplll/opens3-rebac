"""ReBAC storage interfaces"""
from abc import ABC, abstractmethod
from typing import List, Protocol
from internal.types import Tuple

class GraphStore(Protocol):
    """Contract for graph storage implementations"""
    
    @abstractmethod
    def write_tuple(self, tuple_: Tuple) -> bool:
        """Write relationship tuple to graph"""
        ...
    
    @abstractmethod
    def read_tuples(self, subject: str) -> List[Tuple]:
        """Read all outgoing tuples for subject"""
        ...
    
    @abstractmethod
    def delete_tuple(self, tuple_: Tuple) -> bool:
        """Delete a relationship tuple from the graph. Returns True if it existed."""
        ...

    @abstractmethod
    def check(self, subject: str, action: str, object: str) -> bool:
        """Check if subject can perform action on object"""
        ...
    
    @abstractmethod
    def close(self) -> None:
        """Close storage connection"""
        ...

