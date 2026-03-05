"""Skip integration tests when Neo4j is not available."""
import os

import pytest
from neo4j import GraphDatabase


@pytest.fixture(scope="module", autouse=True)
def require_neo4j():
    uri = os.environ.get("NEO4J_URI", "bolt://localhost:7687")
    try:
        driver = GraphDatabase.driver(
            uri,
            auth=("neo4j", os.environ.get("NEO4J_PASSWORD", "password123")),
        )
        driver.verify_connectivity()
        driver.close()
    except Exception as e:
        pytest.skip(f"Neo4j not available: {e}")
