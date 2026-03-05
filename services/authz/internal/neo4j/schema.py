"""Neo4j schema: Node labels, relationship types, permission levels"""
from enum import Enum
from typing import List, Dict

class NodeLabel(str, Enum):
    """Graph node labels"""
    USER = "User"
    GROUP = "Group"
    DOCUMENT = "Document"
    FOLDER = "Folder"
    RESOURCE = "Resource"


class RelationType(str, Enum):
    """ReBAC relationship types"""
    MEMBER_OF = "MEMBER_OF"
    HAS_PERMISSION = "HAS_PERMISSION"
    # Legacy / optional direct relations (for backward compatibility)
    OWNER_OF = "OWNER_OF"
    VIEWER = "VIEWER"
    PARENT_OF = "PARENT_OF"


class PermissionLevel(str, Enum):
    """Permission level for HAS_PERMISSION relation (stored as relationship property)."""
    READ = "read"
    WRITE = "write"
    CREATE = "create"   # e.g. CreateBucket
    DELETE = "delete"
    ADMIN = "admin"


# Action (from Check request) -> which permission levels grant that action.
# Higher level implies lower: admin grants everything, read grants only read.
ALLOWED_LEVELS_PER_ACTION: Dict[str, List[str]] = {
    "read": [PermissionLevel.READ.value, PermissionLevel.WRITE.value, PermissionLevel.CREATE.value, PermissionLevel.DELETE.value, PermissionLevel.ADMIN.value],
    "write": [PermissionLevel.WRITE.value, PermissionLevel.CREATE.value, PermissionLevel.DELETE.value, PermissionLevel.ADMIN.value],
    "create": [PermissionLevel.CREATE.value, PermissionLevel.DELETE.value, PermissionLevel.ADMIN.value],
    "delete": [PermissionLevel.DELETE.value, PermissionLevel.ADMIN.value],
    "admin": [PermissionLevel.ADMIN.value],
}

# Legacy: action -> relation types (for direct edges without level, if we keep them)
PERMISSION_RULES: Dict[str, List[RelationType]] = {
    "read": [RelationType.VIEWER, RelationType.OWNER_OF],
    "write": [RelationType.OWNER_OF],
    "create": [RelationType.OWNER_OF],
    "delete": [RelationType.OWNER_OF],
    "admin": [RelationType.OWNER_OF],
}


def infer_node_label(entity_id: str) -> NodeLabel:
    """Infer node label from entity ID prefix"""
    if entity_id.startswith("user:"):
        return NodeLabel.USER
    if entity_id.startswith("group:"):
        return NodeLabel.GROUP
    if entity_id.startswith("doc:"):
        return NodeLabel.DOCUMENT
    if entity_id.startswith("folder:"):
        return NodeLabel.FOLDER
    if entity_id.startswith("bucket:"):
        return NodeLabel.RESOURCE
    if entity_id.startswith("object:"):
        return NodeLabel.RESOURCE
    return NodeLabel.RESOURCE


def is_valid_permission_level(level: str) -> bool:
    """Check if string is a valid permission level."""
    return level in {p.value for p in PermissionLevel}
