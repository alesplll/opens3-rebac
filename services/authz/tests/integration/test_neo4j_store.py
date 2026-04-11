"""Integration tests for Neo4jStore: transitive ReBAC and HAS_PERMISSION."""
import os

import pytest

from internal.types import Tuple
from internal.repositories.neo4j.store import Neo4jStore
from internal.repositories.neo4j.schema import RelationType


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


# ── Permission level hierarchy ─────────────────────────────────────────────────

def test_level_write_implies_read(neo4j_store, clean_graph):
    """write level grants read action (write ⊇ read)."""
    neo4j_store.write_tuple(
        Tuple("user:alice", RelationType.HAS_PERMISSION.value, "bucket:photos", level="write")
    )
    assert neo4j_store.check("user:alice", "read", "bucket:photos") is True
    assert neo4j_store.check("user:alice", "write", "bucket:photos") is True
    assert neo4j_store.check("user:alice", "delete", "bucket:photos") is False
    assert neo4j_store.check("user:alice", "admin", "bucket:photos") is False


def test_level_delete_implies_write_and_read(neo4j_store, clean_graph):
    """delete level grants delete, create, write, read — but not admin."""
    neo4j_store.write_tuple(
        Tuple("user:bob", RelationType.HAS_PERMISSION.value, "bucket:data", level="delete")
    )
    assert neo4j_store.check("user:bob", "read", "bucket:data") is True
    assert neo4j_store.check("user:bob", "write", "bucket:data") is True
    assert neo4j_store.check("user:bob", "create", "bucket:data") is True
    assert neo4j_store.check("user:bob", "delete", "bucket:data") is True
    assert neo4j_store.check("user:bob", "admin", "bucket:data") is False


def test_level_create_does_not_imply_delete(neo4j_store, clean_graph):
    """create level grants create, write, read — but NOT delete or admin."""
    neo4j_store.write_tuple(
        Tuple("user:carol", RelationType.HAS_PERMISSION.value, "bucket:shared", level="create")
    )
    assert neo4j_store.check("user:carol", "read", "bucket:shared") is True
    assert neo4j_store.check("user:carol", "write", "bucket:shared") is True
    assert neo4j_store.check("user:carol", "create", "bucket:shared") is True
    assert neo4j_store.check("user:carol", "delete", "bucket:shared") is False
    assert neo4j_store.check("user:carol", "admin", "bucket:shared") is False


def test_level_read_denies_everything_else(neo4j_store, clean_graph):
    """read level grants only read."""
    neo4j_store.write_tuple(
        Tuple("user:viewer", RelationType.HAS_PERMISSION.value, "bucket:public", level="read")
    )
    assert neo4j_store.check("user:viewer", "read", "bucket:public") is True
    assert neo4j_store.check("user:viewer", "write", "bucket:public") is False
    assert neo4j_store.check("user:viewer", "create", "bucket:public") is False
    assert neo4j_store.check("user:viewer", "delete", "bucket:public") is False
    assert neo4j_store.check("user:viewer", "admin", "bucket:public") is False


# ── Legacy relations: OWNER_OF, VIEWER ────────────────────────────────────────

def test_owner_of_grants_all_actions(neo4j_store, clean_graph):
    """OWNER_OF (legacy) grants read, write, create, delete, admin."""
    neo4j_store.write_tuple(
        Tuple("user:owner", RelationType.OWNER_OF.value, "bucket:mine")
    )
    for action in ("read", "write", "create", "delete", "admin"):
        assert neo4j_store.check("user:owner", action, "bucket:mine") is True, \
            f"OWNER_OF should grant {action}"


def test_viewer_grants_only_read(neo4j_store, clean_graph):
    """VIEWER (legacy) grants only read."""
    neo4j_store.write_tuple(
        Tuple("user:guest", RelationType.VIEWER.value, "bucket:docs")
    )
    assert neo4j_store.check("user:guest", "read", "bucket:docs") is True
    assert neo4j_store.check("user:guest", "write", "bucket:docs") is False
    assert neo4j_store.check("user:guest", "delete", "bucket:docs") is False


# ── S3 entity ID format ────────────────────────────────────────────────────────

def test_s3_bucket_and_object_ids(neo4j_store, clean_graph):
    """Real S3-format IDs: bucket:photos, object:photos/cat.jpg."""
    neo4j_store.write_tuple(
        Tuple("user:alice", RelationType.HAS_PERMISSION.value, "bucket:photos", level="write")
    )
    # PutObject: check is on parent bucket
    assert neo4j_store.check("user:alice", "write", "bucket:photos") is True

    # GetObject: check is on specific object
    neo4j_store.write_tuple(
        Tuple("user:alice", RelationType.HAS_PERMISSION.value, "object:photos/cat.jpg", level="read")
    )
    assert neo4j_store.check("user:alice", "read", "object:photos/cat.jpg") is True
    assert neo4j_store.check("user:alice", "delete", "object:photos/cat.jpg") is False


def test_s3_owner_creates_bucket(neo4j_store, clean_graph):
    """After CreateBucket, Gateway writes OWNER_OF: user gets full access."""
    neo4j_store.write_tuple(
        Tuple("user:alice", RelationType.OWNER_OF.value, "bucket:my-bucket")
    )
    assert neo4j_store.check("user:alice", "read", "bucket:my-bucket") is True
    assert neo4j_store.check("user:alice", "delete", "bucket:my-bucket") is True
    assert neo4j_store.check("user:bob", "read", "bucket:my-bucket") is False


# ── PARENT_OF ─────────────────────────────────────────────────────────────────

def test_parent_of_written_to_graph(neo4j_store, clean_graph):
    """PARENT_OF can be written. NOTE: check() does not traverse PARENT_OF,
    so bucket permissions are NOT inherited by objects automatically.
    Gateway must write explicit permissions on the object after PutObject."""
    neo4j_store.write_tuple(
        Tuple("bucket:photos", RelationType.PARENT_OF.value, "object:photos/cat.jpg")
    )
    tuples = neo4j_store.read_tuples("bucket:photos")
    assert any(
        t.relation == "PARENT_OF" and t.object == "object:photos/cat.jpg"
        for t in tuples
    )


def test_parent_of_does_not_propagate_permissions(neo4j_store, clean_graph):
    """Bucket-level permission is NOT inherited by child objects via PARENT_OF.
    This is by design: check() only traverses MEMBER_OF + HAS_PERMISSION.
    Object permissions must be set explicitly."""
    neo4j_store.write_tuple(
        Tuple("user:alice", RelationType.HAS_PERMISSION.value, "bucket:photos", level="read")
    )
    neo4j_store.write_tuple(
        Tuple("bucket:photos", RelationType.PARENT_OF.value, "object:photos/cat.jpg")
    )
    # alice has read on bucket, but NOT automatically on the object
    assert neo4j_store.check("user:alice", "read", "object:photos/cat.jpg") is False
