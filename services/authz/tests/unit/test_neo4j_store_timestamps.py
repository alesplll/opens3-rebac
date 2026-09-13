"""Unit tests for where graph timestamps come from: the database, not the service."""
import re
from pathlib import Path
import sys

REPO_ROOT = Path(__file__).resolve().parents[4]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from internal.types import Tuple
from internal.repositories.neo4j.schema import RelationType
from internal.repositories.neo4j.store import Neo4jStore


class _FakeResult:
    def single(self):
        return {"written": True}


class _FakeSession:
    def __init__(self, recorder):
        self._recorder = recorder

    def __enter__(self):
        return self

    def __exit__(self, *exc_info):
        return False

    def run(self, query, **params):
        self._recorder.append((query, params))
        return _FakeResult()


class _FakeDriver:
    """Records every statement instead of talking to Neo4j."""

    def __init__(self):
        self.statements = []

    def session(self):
        return _FakeSession(self.statements)


def _store_with(driver):
    """Wire a store to a fake driver; __init__ would open a real connection."""
    store = Neo4jStore.__new__(Neo4jStore)
    store.driver = driver
    return store


_TIMESTAMP_ASSIGNMENT = re.compile(r"\.(?:created_at|updated_at)\s*=\s*([^\s,]+)")


def _timestamp_sources(query):
    """Every expression a timestamp property is assigned from."""
    return set(_TIMESTAMP_ASSIGNMENT.findall(query))


class TestGraphTimestampsComeFromTheDatabase:
    def test_permission_write_stamps_time_in_cypher(self):
        driver = _FakeDriver()
        store = _store_with(driver)

        store.write_tuple(
            Tuple("user:alex", RelationType.HAS_PERMISSION.value, "bucket:photos", level="admin")
        )

        query, params = driver.statements[0]
        assert _timestamp_sources(query) == {"timestamp()"}
        assert "now" not in params

    def test_plain_relation_write_stamps_time_in_cypher(self):
        driver = _FakeDriver()
        store = _store_with(driver)

        store.write_tuple(Tuple("user:alex", RelationType.MEMBER_OF.value, "group:devops"))

        query, params = driver.statements[0]
        assert _timestamp_sources(query) == {"timestamp()"}
        assert "now" not in params
