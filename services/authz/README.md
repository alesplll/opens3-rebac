<p align="center">
  <img src="https://img.shields.io/badge/Python-3.12-3776AB?logo=python&logoColor=white" alt="Python" />
  <img src="https://img.shields.io/badge/gRPC-Enabled-00C7B7?logo=google-cloud&logoColor=white" alt="gRPC" />
  <img src="https://img.shields.io/badge/Neo4j-Graph%20DB-008CC1?logo=neo4j&logoColor=white" alt="Neo4j" />
  <img src="https://img.shields.io/badge/Redis-Cache-DC382D?logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/Kafka-Audit%20Stream-231F20?logo=apache-kafka&logoColor=white" alt="Kafka" />
  <img src="https://img.shields.io/badge/Docker-Containerized-2496ED?logo=docker&logoColor=white" alt="Docker" />
</p>

<p align="center">
  <strong>ReBAC Auth Service</strong> · Relationship-Based Authorization Engine
</p>

---

# ReBAC Auth Service

Centralized **relationship-based access control (ReBAC)** engine exposed via gRPC. Other services ask:

> "Can `user:alice` **read** resource `doc:123`?" → `ALLOW` / `DENY`

Authorization decisions are made by traversing a **graph** of users, groups, and resources stored in Neo4j. Results are cached in Redis; every decision and relationship change is audited via Kafka.

---

## Architecture

```
gRPC client
    │  Check(subject, action, object)
    ▼
PermissionService
    ├── Redis cache  →  hit: return cached decision
    └── Neo4j graph  →  miss: traverse graph, cache result, return
             │
             └── Kafka  →  audit event (ACCESS_GRANTED / ACCESS_DENIED)
                           + cache invalidation on WriteTuple / DeleteTuple
```

### Graph Model

**Nodes:** `User` · `Group` · `Resource` (prefix determines label: `user:`, `group:`, `resource:`, `bucket:`, `object:`, …)

**Edges:**

| Relation | From → To | Meaning |
|---|---|---|
| `MEMBER_OF` | User/Group → Group | Group membership; transitive (`*0..`) |
| `HAS_PERMISSION` | User/Group → Resource | Permission with `level`: `read`/`write`/`create`/`delete`/`admin` |
| `OWNER_OF` | User/Group → Resource | Full access (legacy) |
| `VIEWER` | User/Group → Resource | Read access (legacy) |
| `PARENT_OF` | Resource → Resource | Resource hierarchy |

**Level hierarchy:** `admin` ⊇ `delete` ⊇ `create` ⊇ `write` ⊇ `read`

### gRPC API

| RPC | Description |
|---|---|
| `Check(subject, action, object)` | Returns `{allowed: bool}` |
| `WriteTuple(subject, relation, object, level?)` | Add a relationship |
| `DeleteTuple(subject, relation, object)` | Remove a relationship |
| `Read(subject)` | List all outgoing relationships |
| `Health` (standard gRPC health protocol) | Used by K8s liveness/readiness probes |

---

## Quick Start (local dev)

### Prerequisites

- Python 3.12
- Docker & Docker Compose
- [`grpcurl`](https://github.com/fullstorydev/grpcurl) for manual testing

### 1. Setup

```bash
# Clone and enter the project
cd rebac-auth-service

# Create venv and install dependencies
python -m venv venv
source venv/bin/activate
pip install -e ".[test]"

# Generate gRPC stubs from proto
bash proto/generate.sh
```

### 2. Start infrastructure (Neo4j, Redis, Kafka)

```bash
docker compose -f deploy/local/docker-compose.yml up -d
```

- **Neo4j Browser:** http://localhost:7474 — login `neo4j` / `password123`
- **Redis:** `redis-cli ping` → `PONG`
- **Kafka:** `localhost:9092`

### 3. Run the gRPC server

```bash
source venv/bin/activate
python entrypoints/server/main.py
# → ReBAC Auth Service gRPC :50051
```

### 4. (Optional) Run cache invalidator

In a separate terminal — consumes Kafka events and invalidates Redis cache on relationship changes:

```bash
source venv/bin/activate
python entrypoints/cache_invalidator.py
```

### Environment variables

All parameters have sensible localhost defaults; override for Docker/K8s:

| Variable | Default | Docker default |
|---|---|---|
| `NEO4J_URI` | `bolt://localhost:7687` | `bolt://neo4j:7687` |
| `NEO4J_USER` | `neo4j` | `neo4j` |
| `NEO4J_PASSWORD` | `password123` | `password123` |
| `REDIS_HOST` | `localhost` | `redis` |
| `REDIS_PORT` | `6379` | `6379` |
| `KAFKA_BOOTSTRAP` | `localhost:9092` | `kafka:29092` |
| `GRPC_PORT` | `50051` | `50051` |

---

## Docker

### Build and run as a container

```bash
# Build image
docker build -t rebac-auth-service .

# Run (against already-running local infra)
docker run --rm --network host rebac-auth-service
```

### Full stack (infra + service)

```bash
docker compose -f deploy/local/docker-compose.yml --profile app up -d
```

### Stop everything

```bash
docker compose -f deploy/local/docker-compose.yml --profile app down

# To also remove volumes (wipes Neo4j data):
docker compose -f deploy/local/docker-compose.yml --profile app down -v
```

### Rebuild after code changes

```bash
docker compose -f deploy/local/docker-compose.yml --profile app build rebac-auth-service
docker compose -f deploy/local/docker-compose.yml --profile app up -d rebac-auth-service
```

---

## Manual Testing (grpcurl)

All commands below assume the server is running on `localhost:50051`.

### Verify service is up

```bash
grpcurl -plaintext localhost:50051 list
# → rebac.authz.v1.PermissionService
# → grpc.health.v1.Health
```

### Health check

```bash
grpcurl -plaintext -d '{"service": "rebac.authz.v1.PermissionService"}' \
  localhost:50051 grpc.health.v1.Health/Check
# → {"status": "SERVING"}
```

### Write relationships

```bash
# alex is a member of devops
grpcurl -plaintext -d '{"subject":"user:alex","relation":"MEMBER_OF","object":"group:devops"}' \
  localhost:50051 rebac.authz.v1.PermissionService/WriteTuple

# devops has admin permission on server-1
grpcurl -plaintext -d '{"subject":"group:devops","relation":"HAS_PERMISSION","object":"resource:server-1","level":"admin"}' \
  localhost:50051 rebac.authz.v1.PermissionService/WriteTuple

# viewers group has read-only on doc1
grpcurl -plaintext -d '{"subject":"group:viewers","relation":"HAS_PERMISSION","object":"resource:doc1","level":"read"}' \
  localhost:50051 rebac.authz.v1.PermissionService/WriteTuple

# bob is a viewer
grpcurl -plaintext -d '{"subject":"user:bob","relation":"MEMBER_OF","object":"group:viewers"}' \
  localhost:50051 rebac.authz.v1.PermissionService/WriteTuple

# Transitive chain: alice → payments → finance → billing (read)
grpcurl -plaintext -d '{"subject":"user:alice","relation":"MEMBER_OF","object":"group:payments"}' \
  localhost:50051 rebac.authz.v1.PermissionService/WriteTuple
grpcurl -plaintext -d '{"subject":"group:payments","relation":"MEMBER_OF","object":"group:finance"}' \
  localhost:50051 rebac.authz.v1.PermissionService/WriteTuple
grpcurl -plaintext -d '{"subject":"group:finance","relation":"HAS_PERMISSION","object":"resource:billing","level":"read"}' \
  localhost:50051 rebac.authz.v1.PermissionService/WriteTuple
```

### Check permissions

```bash
# alex: admin implies read on server-1 → ALLOW
grpcurl -plaintext -d '{"subject":"user:alex","action":"admin","object":"resource:server-1"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check

grpcurl -plaintext -d '{"subject":"user:alex","action":"read","object":"resource:server-1"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check

# eve has no rights → DENY
grpcurl -plaintext -d '{"subject":"user:eve","action":"read","object":"resource:server-1"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check

# bob: read allowed, write denied
grpcurl -plaintext -d '{"subject":"user:bob","action":"read","object":"resource:doc1"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check
grpcurl -plaintext -d '{"subject":"user:bob","action":"write","object":"resource:doc1"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check

# alice: transitive chain alice→payments→finance → read on billing, but not write
grpcurl -plaintext -d '{"subject":"user:alice","action":"read","object":"resource:billing"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check
grpcurl -plaintext -d '{"subject":"user:alice","action":"write","object":"resource:billing"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check
```

### Delete relationships

```bash
# Remove alex from devops → loses access to server-1
grpcurl -plaintext -d '{"subject":"user:alex","relation":"MEMBER_OF","object":"group:devops"}' \
  localhost:50051 rebac.authz.v1.PermissionService/DeleteTuple
# → {"success": true}

# Verify access is gone
grpcurl -plaintext -d '{"subject":"user:alex","action":"read","object":"resource:server-1"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Check
# → {"allowed": false}

# Deleting a non-existent tuple returns false
grpcurl -plaintext -d '{"subject":"user:nobody","relation":"MEMBER_OF","object":"group:nobody"}' \
  localhost:50051 rebac.authz.v1.PermissionService/DeleteTuple
# → {"success": false}
```

### Read relationships

```bash
grpcurl -plaintext -d '{"subject":"user:alex"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Read

grpcurl -plaintext -d '{"subject":"group:devops"}' \
  localhost:50051 rebac.authz.v1.PermissionService/Read
```

### Kafka audit events

```bash
docker exec -it $(docker ps -qf name=kafka) \
  kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic auth-changes \
  --from-beginning
```

Events emitted:
- `tuple_written` / `tuple_removed` — on WriteTuple / DeleteTuple (includes `invalidation_hints` for Redis)
- `ACCESS_GRANTED` / `ACCESS_DENIED` — on every Check

### Redis cache

```bash
# Check cached keys
redis-cli KEYS "auth_decision:*"

# Inspect a value (1 = allowed, 0 = denied)
redis-cli GET "auth_decision:user:alex:read:resource:server-1"

# TTL (default 30s)
redis-cli TTL "auth_decision:user:alex:read:resource:server-1"
```

After a `DeleteTuple` + Kafka processing, the corresponding key is invalidated automatically (requires `cache_invalidator` running).

### Neo4j graph (Browser)

Open http://localhost:7474, login `neo4j` / `password123`, run Cypher:

```cypher
-- Full graph
MATCH (n)-[r]->(m) RETURN n, r, m

-- Tabular view
MATCH (a)-[r]->(b) RETURN a.id AS from, type(r) AS relation, b.id AS to

-- HAS_PERMISSION with levels
MATCH (a)-[r:HAS_PERMISSION]->(b) RETURN a.id AS subject, r.level AS level, b.id AS object

-- All nodes by type
MATCH (n) RETURN labels(n)[0] AS type, n.id AS id

-- Wipe everything (start fresh)
MATCH (n) DETACH DELETE n
```

---

## Tests

```bash
source venv/bin/activate

# Unit tests (no infrastructure required)
pytest tests/unit -v

# Unit tests with coverage
pytest tests/unit -v --cov=internal --cov=entrypoints --cov-report=term-missing

# Integration tests (Neo4j must be running)
pytest tests/integration -v -m integration
```

Integration tests auto-skip if Neo4j is unreachable. Run Neo4j first:

```bash
docker compose -f deploy/local/docker-compose.yml up -d neo4j
```

---

## Proto regeneration

After modifying `proto/authz.proto`:

```bash
bash proto/generate.sh
# Regenerates internal/gen/authz_pb2.py and authz_pb2_grpc.py
```

---

## Project structure

```
rebac-auth-service/
├── entrypoints/
│   ├── server/main.py          # gRPC server entry point
│   └── cache_invalidator.py    # Kafka consumer for Redis invalidation
├── internal/
│   ├── rebac/model.py          # PermissionService — orchestrates store/cache/audit
│   ├── neo4j/
│   │   ├── store.py            # Neo4j graph traversal (transitive + HAS_PERMISSION)
│   │   └── schema.py           # Node labels, relation types, level hierarchy
│   ├── cache/
│   │   ├── redis_cache.py      # Redis decision cache
│   │   └── invalidation_consumer.py  # SCAN+DEL invalidation via Kafka
│   ├── kafka/producer.py       # Audit event producer
│   └── types.py                # Tuple dataclass
├── proto/authz.proto           # gRPC contract
├── internal/gen/               # Auto-generated stubs (do not edit)
├── tests/
│   ├── unit/                   # Mocked unit tests
│   └── integration/            # Real Neo4j integration tests
├── deploy/local/docker-compose.yml
├── Dockerfile
└── .github/workflows/ci.yml    # CI: unit + integration tests
```
