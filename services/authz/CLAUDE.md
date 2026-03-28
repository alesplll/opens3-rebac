# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Relationship-Based Access Control (ReBAC) authorization engine exposed via gRPC. Clients call `Check(subject, action, object)` and get back ALLOW/DENY. Authorization decisions are made by traversing a graph of relationships stored in Neo4j.

## Commands

### Setup
```bash
python -m venv venv && source venv/bin/activate
pip install -e ".[test]"
bash proto/generate.sh          # regenerate gRPC stubs into internal/gen/
```

### Infrastructure (Neo4j, Redis, Kafka)
```bash
docker compose -f deploy/local/docker-compose.yml up -d
```

### Run Server
```bash
python entrypoints/server/main.py   # gRPC on :50051
```

### Tests
```bash
pytest tests/unit -v                                    # unit tests (no infra needed)
pytest tests/unit/test_permission_service.py::TestPermissionServiceCheck::test_no_store_denies  # single test
pytest tests/integration -v -m integration              # integration tests (needs Neo4j)
pytest tests/unit -v --cov=internal --cov=entrypoints --cov-report=term-missing  # with coverage
```

Integration tests require Neo4j running and these env vars (defaults work with docker-compose):
- `NEO4J_URI` (default: `bolt://localhost:7687`)
- `NEO4J_USER` (default: `neo4j`)
- `NEO4J_PASSWORD` (default: `password123`)

### Proto Regeneration
After modifying `proto/authz.proto`, run `bash proto/generate.sh`. This generates `internal/gen/authz_pb2.py` and `authz_pb2_grpc.py` with fixed relative imports.

## Architecture

### Layered Structure
- **`entrypoints/`** — Process entry points. `server/main.py` is the gRPC server; `cache_invalidator.py` runs the Kafka consumer for cache invalidation.
- **`internal/rebac/`** — Core domain. `model.py` has `PermissionService` which orchestrates store, cache, and audit. `interfaces.py` defines the `GraphStore` protocol.
- **`internal/neo4j/`** — Neo4j implementation of `GraphStore`. `schema.py` defines node labels, relation types, permission levels, and the entity ID prefix convention. `store.py` implements graph traversal queries.
- **`internal/cache/`** — `DecisionCache` ABC in `interfaces.py`, Redis implementation in `redis_cache.py`, Kafka-driven invalidation in `invalidation_consumer.py`.
- **`internal/kafka/`** — `AuditProducer` emits `tuple_written` events to the `auth-changes` topic.
- **`internal/types.py`** — `Tuple` dataclass: the fundamental unit `(subject, relation, object, level?)`.
- **`proto/`** — gRPC contract. `authz.proto` defines `PermissionService` with `Check`, `WriteTuple`, `Read` RPCs.
- **`internal/gen/`** — Auto-generated gRPC Python stubs. Do not edit manually.

### Authorization Model
Permission checks follow two paths in `Neo4jStore.check()`:
1. **Transitive HAS_PERMISSION** — `(subject)-[:MEMBER_OF*0..]->(x)-[:HAS_PERMISSION {level}]->(resource)`. The `level` property on HAS_PERMISSION edges encodes granularity (read/write/create/delete/admin). Higher levels imply lower ones (admin grants everything). Mapping is in `schema.py:ALLOWED_LEVELS_PER_ACTION`.
2. **Legacy direct relations** — Fallback to direct OWNER_OF/VIEWER edges via `schema.py:PERMISSION_RULES`.

### Entity ID Convention
Entity IDs use a `prefix:name` format (e.g., `user:alice`, `group:devops`, `doc:invoice-42`). The prefix determines the Neo4j node label via `schema.py:infer_node_label()`.

### Check Flow
`gRPC request → PermissionServiceServicer → PermissionService.check() → Redis cache lookup → Neo4j graph query → cache write → response`. On `WriteTuple`, an audit event is emitted to Kafka, and a separate `CacheInvalidationConsumer` process invalidates affected Redis keys.

### Testing Patterns
Unit tests mock `GraphStore`, `DecisionCache`, and `AuditProducer` via `unittest.mock.MagicMock`. Integration tests use a real Neo4j instance and auto-skip if Neo4j is unavailable. Integration tests are marked with `@pytest.mark.integration`.

## Build System
Uses `hatchling` as the build backend. The two packages shipped are `internal` and `entrypoints`. Console script entry point: `rebac-server` → `entrypoints.server.main:run_server`.
