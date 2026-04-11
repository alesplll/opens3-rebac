"""Neo4j GraphStore implementation with transitive ReBAC and HAS_PERMISSION levels."""
from neo4j import GraphDatabase
from typing import List
from internal.types import Tuple
from internal.repositories.neo4j.schema import (
    infer_node_label,
    RelationType,
    ALLOWED_LEVELS_PER_ACTION,
    PERMISSION_RULES,
    is_valid_permission_level,
)

from shared.pkg.py_kit import logger


class Neo4jStore:
    """Neo4j-backed GraphStore with transitive User->Group*->Resource checks."""

    def __init__(self, uri: str, user: str, password: str):
        self.driver = GraphDatabase.driver(uri, auth=(user, password))
        logger.info({}, "Connected to Neo4j", uri=uri)

    def write_tuple(self, tuple_: Tuple) -> bool:
        """Create nodes and relationship in Neo4j. For HAS_PERMISSION, level is required."""
        if tuple_.relation == RelationType.HAS_PERMISSION.value:
            if not tuple_.level or not is_valid_permission_level(tuple_.level):
                logger.warn({}, "HAS_PERMISSION requires valid level", permission_level=tuple_.level)
                return False
            return self._write_has_permission(tuple_)
        return self._write_plain_relation(tuple_)

    def _write_has_permission(self, tuple_: Tuple) -> bool:
        """Create (subject)-[:HAS_PERMISSION {level: ...}]->(object)."""
        s_label = infer_node_label(tuple_.subject).value
        o_label = infer_node_label(tuple_.object).value
        query = """
        MERGE (subject:`%s` {id: $subject_id})
        MERGE (object:`%s` {id: $object_id})
        MERGE (subject)-[r:HAS_PERMISSION]->(object)
        SET r.level = $level
        RETURN r
        """ % (s_label, o_label)
        with self.driver.session() as session:
            result = session.run(
                query,
                subject_id=tuple_.subject,
                object_id=tuple_.object,
                level=tuple_.level,
            )
            return result.single() is not None

    def _write_plain_relation(self, tuple_: Tuple) -> bool:
        """Create (subject)-[:REL_TYPE]->(object) for MEMBER_OF, OWNER_OF, etc."""
        s_label = infer_node_label(tuple_.subject).value
        o_label = infer_node_label(tuple_.object).value
        query = """
        MERGE (subject:`%s` {id: $subject_id})
        MERGE (object:`%s` {id: $object_id})
        MERGE (subject)-[rel:`%s`]->(object)
        RETURN rel
        """ % (s_label, o_label, tuple_.relation)
        logger.debug({}, "Neo4j write plain relation", tuple=str(tuple_))
        with self.driver.session() as session:
            result = session.run(
                query,
                subject_id=tuple_.subject,
                object_id=tuple_.object,
            )
            return result.single() is not None

    def read_tuples(self, subject: str) -> List[Tuple]:
        """Read all outgoing relationships for subject; include level for HAS_PERMISSION."""
        query = """
        MATCH (subject {id: $subject_id})-[rel]->(target)
        RETURN subject.id AS subject_id, type(rel) AS relation, target.id AS object_id, rel.level AS level
        """
        result: List[Tuple] = []
        with self.driver.session() as session:
            records = session.run(query, subject_id=subject)
            for record in records:
                level = record.get("level")
                if level is not None:
                    level = str(level)
                result.append(
                    Tuple(
                        subject=record["subject_id"],
                        relation=record["relation"],
                        object=record["object_id"],
                        level=level,
                    )
                )
        return result

    def delete_tuple(self, tuple_: Tuple) -> bool:
        """Delete a relationship edge. Returns True if it existed and was deleted."""
        query = """
        MATCH (subject {id: $subject_id})-[rel:`%s`]->(object {id: $object_id})
        DELETE rel
        RETURN true AS deleted
        """ % tuple_.relation
        with self.driver.session() as session:
            result = session.run(
                query,
                subject_id=tuple_.subject,
                object_id=tuple_.object,
            )
            rec = result.single()
            deleted = bool(rec and rec["deleted"])
            logger.debug({}, "Neo4j delete_tuple", tuple=str(tuple_), deleted=deleted)
            return deleted

    def check(self, subject: str, action: str, object: str) -> bool:
        """
        Check authorization via transitive path:
        (subject)-[:MEMBER_OF*0..]->(user or group)-[:HAS_PERMISSION]->(resource)
        with permission level sufficient for action.
        """
        allowed_levels = ALLOWED_LEVELS_PER_ACTION.get(action)
        if allowed_levels:
            query = """
            MATCH (subj {id: $subject})-[:MEMBER_OF*0..]->(x)-[p:HAS_PERMISSION]->(res {id: $object})
            WHERE p.level IN $allowed_levels
            RETURN count(p) > 0 AS authorized
            """
            with self.driver.session() as session:
                r = session.run(
                    query,
                    subject=subject,
                    object=object,
                    allowed_levels=allowed_levels,
                )
                rec = r.single()
                if rec and rec["authorized"]:
                    logger.debug({}, "Neo4j transitive check: authorized")
                    return True

        # Fallback: direct relation (legacy OWNER_OF, VIEWER)
        allowed_rels = PERMISSION_RULES.get(action, [])
        if not allowed_rels:
            logger.warn({}, "Unknown action", action=action)
            return False

        query = """
        MATCH (subject {id: $subject})-[rel]->(target {id: $object})
        WHERE type(rel) IN $allowed_rels
        RETURN count(rel) > 0 AS authorized
        """
        with self.driver.session() as session:
            r = session.run(
                query,
                subject=subject,
                object=object,
                allowed_rels=[r.value for r in allowed_rels],
            )
            rec = r.single()
            authorized = rec["authorized"] if rec else False
            logger.debug({}, "Neo4j direct check result", authorized=authorized)
            return authorized

    def health(self) -> None:
        """Raise if Neo4j is unreachable."""
        self.driver.verify_connectivity()

    def close(self) -> None:
        self.driver.close()
        logger.info({}, "Neo4j connection closed")
