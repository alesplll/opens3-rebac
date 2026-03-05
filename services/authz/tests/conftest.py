"""Pytest configuration and shared fixtures."""
import os

import pytest


def pytest_configure(config):
    config.addinivalue_line("markers", "integration: mark test as integration (requires Neo4j)")


@pytest.fixture
def neo4j_uri():
    return os.environ.get("NEO4J_URI", "bolt://localhost:7687")


@pytest.fixture
def neo4j_available(neo4j_uri):
    """Skip if Neo4j is not reachable (for integration tests)."""
    try:
        from neo4j import GraphDatabase
        driver = GraphDatabase.driver(neo4j_uri, auth=("neo4j", os.environ.get("NEO4J_PASSWORD", "password123")))
        driver.verify_connectivity()
        driver.close()
        return True
    except Exception:
        return False
