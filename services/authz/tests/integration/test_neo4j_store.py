"""Integration tests for Neo4jStore: transitive ReBAC and HAS_PERMISSION."""
import os

import pytest

from internal.types import Tuple
from internal.neo4j.store import Neo4jStore
from internal.neo4j.schema import RelationType


pytestmark = pytest.mark.integration


@pytest.fixture(scope="module")
def neo4j_store():
    uri = os.environ.get("NEO4J_URI", "bolt://localhost:7687")
    user = os.environ.get("NEO4J_USER", "neo4j")
    password = os.environ.get("NEO4J_PASSWORD", "password123")
    store = Neo4jStore(uri=uri, user=user, password=password)
    yield store
    store.close()


@pytest.fixture
def clean_graph(neo4j_store):
    """Clear all nodes and relationships before test (optional, use with care)."""
    with neo4j_store.driver.session() as session:
        session.run("MATCH (n) DETACH DELETE n")
    yield
    with neo4j_store.driver.session() as session:
        session.run("MATCH (n) DETACH DELETE n")


def test_write_member_of_and_has_permission(neo4j_store, clean_graph):
    """Alex MEMBER_OF DevOps, DevOps HAS_PERMISSION(admin) Server-1."""
    assert neo4j_store.write_tuple(
        Tuple("user:alex", "MEMBER_OF", "group:devops", level=None)
    ) is True
    assert neo4j_store.write_tuple(
        Tuple("group:devops", RelationType.HAS_PERMISSION.value, "resource:server-1", level="admin")
    ) is True

    # Transitive: Alex can admin Server-1
    assert neo4j_store.check("user:alex", "admin", "resource:server-1") is True
    assert neo4j_store.check("user:alex", "read", "resource:server-1") is True
    assert neo4j_store.check("user:alex", "delete", "resource:server-1") is True


def test_write_has_permission_requires_level(neo4j_store, clean_graph):
    """HAS_PERMISSION without level should fail (or be rejected)."""
    # Store returns False when level is missing for HAS_PERMISSION
    result = neo4j_store.write_tuple(
        Tuple("group:g1", RelationType.HAS_PERMISSION.value, "resource:r1", level=None)
    )
    assert result is False


def test_read_tuples_returns_level(neo4j_store, clean_graph):
    neo4j_store.write_tuple(
        Tuple("group:devs", RelationType.HAS_PERMISSION.value, "resource:repo1", level="write")
    )
    tuples = neo4j_store.read_tuples("group:devs")
    assert len(tuples) >= 1
    has_perm = next((t for t in tuples if t.relation == "HAS_PERMISSION" and t.object == "resource:repo1"), None)
    assert has_perm is not None
    assert has_perm.level == "write"


def test_transitive_chain_multiple_groups(neo4j_store, clean_graph):
    """user:alice -> group:payments -> group:finance (nested) -> resource:billing (read)."""
    neo4j_store.write_tuple(Tuple("user:alice", "MEMBER_OF", "group:payments", level=None))
    neo4j_store.write_tuple(Tuple("group:payments", "MEMBER_OF", "group:finance", level=None))
    neo4j_store.write_tuple(
        Tuple("group:finance", RelationType.HAS_PERMISSION.value, "resource:billing", level="read")
    )

    # Alice in payments, payments in finance, finance has read on billing
    assert neo4j_store.check("user:alice", "read", "resource:billing") is True
    assert neo4j_store.check("user:alice", "write", "resource:billing") is False


def test_direct_user_has_permission(neo4j_store, clean_graph):
    """User can have HAS_PERMISSION directly (0 steps MEMBER_OF)."""
    neo4j_store.write_tuple(
        Tuple("user:bob", RelationType.HAS_PERMISSION.value, "resource:doc1", level="read")
    )
    assert neo4j_store.check("user:bob", "read", "resource:doc1") is True
    assert neo4j_store.check("user:bob", "write", "resource:doc1") is False


def test_delete_tuple_removes_relationship(neo4j_store, clean_graph):
    """delete_tuple removes the edge; read_tuples no longer returns it."""
    t = Tuple("user:alice", "MEMBER_OF", "group:dev", level=None)
    neo4j_store.write_tuple(t)

    assert any(x.relation == "MEMBER_OF" for x in neo4j_store.read_tuples("user:alice"))

    result = neo4j_store.delete_tuple(t)

    assert result is True
    assert not any(x.relation == "MEMBER_OF" for x in neo4j_store.read_tuples("user:alice"))


def test_delete_nonexistent_tuple_returns_false(neo4j_store, clean_graph):
    """Deleting a non-existent edge returns False."""
    result = neo4j_store.delete_tuple(
        Tuple("user:nobody", "MEMBER_OF", "group:nothing")
    )
    assert result is False


def test_check_returns_false_after_membership_deleted(neo4j_store, clean_graph):
    """Access is denied after the MEMBER_OF tuple is removed."""
    neo4j_store.write_tuple(Tuple("user:alice", "MEMBER_OF", "group:ops", level=None))
    neo4j_store.write_tuple(
        Tuple("group:ops", RelationType.HAS_PERMISSION.value, "resource:server", level="read")
    )
    assert neo4j_store.check("user:alice", "read", "resource:server") is True

    neo4j_store.delete_tuple(Tuple("user:alice", "MEMBER_OF", "group:ops"))
    assert neo4j_store.check("user:alice", "read", "resource:server") is False


def test_delete_has_permission_tuple(neo4j_store, clean_graph):
    """Deleting a HAS_PERMISSION edge revokes access."""
    perm = Tuple("group:eng", RelationType.HAS_PERMISSION.value, "resource:ci", level="write")
    neo4j_store.write_tuple(perm)
    assert neo4j_store.check("group:eng", "write", "resource:ci") is True

    neo4j_store.delete_tuple(Tuple("group:eng", RelationType.HAS_PERMISSION.value, "resource:ci"))
    assert neo4j_store.check("group:eng", "write", "resource:ci") is False
