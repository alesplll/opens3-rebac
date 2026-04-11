# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Relationship-Based Access Control (ReBAC) authorization engine exposed via gRPC. Clients call `Check(subject, action, object)` and get back ALLOW/DENY. Authorization decisions are made by traversing a graph of relationships stored in Neo4j.

## Commands

### Setup
```bash
python -m venv venv && source venv/bin/activate
pip install -e ".[test]"
bash proto/generate.sh          # regenerate gRPC stubs into shared/pkg/py/
```

### Infrastructure (Neo4j, Redis, Kafka)
```bash
docker compose -f deploy/local/docker-compose.yml up -d
```

### Run Server
```bash
python entrypoints/server/main.py   # gRPC on :50051
```

### Run Cache Invalidator (separate process)
```bash
python entrypoints/cache_invalidator.py
```

### Tests
```bash
pytest tests/unit -v                                    # unit tests (no infra needed)
pytest tests/integration -v -m integration              # integration tests (needs Neo4j)
pytest tests/unit -v --cov=internal --cov=entrypoints --cov-report=term-missing  # with coverage
```

Integration tests require Neo4j running and these env vars (defaults work with docker-compose):
- `NEO4J_URI` (default: `bolt://localhost:7687`)
- `NEO4J_USER` (default: `neo4j`)
- `NEO4J_PASSWORD` (default: `password123`)

### Proto Regeneration
After modifying `shared/api/authz/v1/authz.proto`, run `bash proto/generate.sh`.

## Architecture

### Layered Structure

```
entrypoints/          — process entry points (transport layer)
  server/
    main.py           — starts gRPC server, wires observability + container
    servicer.py       — gRPC handler: translates protobuf ↔ service calls
  cache_invalidator.py — Kafka consumer process for Redis invalidation

internal/             — all internal code (not importable from outside)
  config.py           — reads env vars ONCE at startup, singleton cfg
  container.py        — dependency injection: wires all objects together
  types.py            — Tuple dataclass (core domain type)

  permission/         — SERVICE layer: business logic
    interfaces.py     — GraphStore Protocol (what the service needs from storage)
    service.py        — PermissionService: orchestrates cache → graph → audit

  repositories/       — REPOSITORY layer: data access
    neo4j/
      schema.py       — node labels, relation types, permission level hierarchy
      store.py        — Neo4jStore: graph traversal queries
    cache/
      interfaces.py   — DecisionCache ABC
      redis_cache.py  — RedisDecisionCache: get/set/health
      invalidation_consumer.py — CacheInvalidationConsumer: SCAN+DEL on Kafka events
    kafka/
      producer.py     — AuditProducer: emits tuple/decision events to Kafka

  metric/             — cross-cutting: Prometheus/OTel metrics
    authz_metrics.py  — counters and histograms for cache, decisions, neo4j latency
```

### Authorization Flow (Check RPC)

```
gRPC request
  → servicer.py: decode enum → string, call service
  → PermissionService.check()
      1. Redis cache lookup (cache.get)
         hit  → return cached decision
         miss → continue
      2. Neo4j graph traversal (store.check)
         MATCH (subject)-[:MEMBER_OF*0..]->(x)-[:HAS_PERMISSION]->(object)
      3. Cache write (cache.set, TTL 30s)
      4. Audit emit (AuditProducer → Kafka auth-audit)
  → servicer.py: wrap in CheckResponse
```

### Cache Invalidation Flow

```
PermissionService.write_tuple() / delete_tuple()
  → AuditProducer.send_tuple_event() → Kafka auth-changes

  (separate process)
CacheInvalidationConsumer.run()
  → poll Kafka auth-changes
  → _handle_event(): extract invalidation_hints patterns
  → Redis SCAN + DELETE matching keys
```

### Entity ID Convention

`prefix:name` format: `user:alice`, `group:devops`, `bucket:photos`, `object:photos/cat.jpg`

### Permission Level Hierarchy

```
read < write < create < delete < admin
```

`HAS_PERMISSION` edge with `level=write` grants: write, read (everything below).

### Testing Patterns

Unit tests mock `GraphStore`, `DecisionCache`, `AuditProducer` via `unittest.mock.MagicMock`.
HealthCheck tests mock `store.health()` and `cache.health()` — NOT internal attrs like `driver` or `_client`.
Integration tests use real Neo4j and auto-skip if unreachable.
