# CLAUDE.md — opens3-rebac

> Context file for AI assistants. Covers actual project state, architecture, and working guidelines.

---

## What this project is

**OpenS3-ReBAC** — a distributed S3-compatible object storage with **ReBAC** (Relationship-Based Access Control) authorization. Team learning project (4 people).

Repo: https://github.com/alesplll/opens3-rebac  
Wiki: https://github.com/alesplll/opens3-rebac/wiki

---

## Services (current actual state)

| Service | Lang | Port | Owner | Status |
|---|---|---|---|---|
| **Gateway** | Go | `:8080` (HTTP) | Max | ⚠️ Not implemented |
| **Auth** | Go | `:50050` (gRPC) | — | ✅ Complete |
| **AuthZ (ReBAC)** | Python | `:50051` (gRPC) | Alexa | ✅ Complete |
| **Metadata** | Go | `:50052` (gRPC) | Anya | ✅ Complete |
| **Storage (Data Node)** | Go | `:50053` (gRPC) | Ilya | ✅ Complete |
| **Users** | Go | `:50054` (gRPC) | — | ✅ Complete |
| **Quota** | Rust | `:50055` (gRPC) | — | ✅ Complete |

Service-to-service communication: **gRPC**. Client → Gateway: **HTTP / S3 API**.

---

## Architecture

```
Client (HTTP / S3 API)
        │
        ▼
   Gateway :8080          ← single entry point (NOT YET IMPLEMENTED)
   /   |    |    \
Auth  AuthZ Meta Storage
:50050 :50051 :50052 :50053
  │      │      │      │
  │    Neo4j  PostgreSQL filesystem
  │    Redis  Kafka
  │
Users :50054
  │
PostgreSQL

Quota :50055  ← enforces per-user/bucket limits (Redis + Kafka)
```

---

## Tech stack

- **Go 1.24.1** — Auth, Users, Metadata, Storage (go.work workspace)
- **Python 3.12** — AuthZ
- **Rust 2021** — Quota (tonic + tokio + fred)
- **gRPC / Protobuf** — all inter-service communication
- **PostgreSQL** — Users (port 5432), Metadata (port 5433)
- **Neo4j** — AuthZ relation graph
- **Redis** — Auth/AuthZ cache, Quota hot-path storage
- **Kafka** — async events (audit, cache invalidation, object lifecycle)
- **Docker Compose** — local dev (`--profile services`, `--profile observability`)
- **OpenTelemetry / Jaeger / Prometheus / Grafana / Kibana** — observability

---

## Repository structure

```
opens3-rebac/
├── shared/
│   ├── api/                   # Proto source files (source of truth)
│   │   ├── auth/v1/auth.proto
│   │   ├── user/v1/user.proto
│   │   ├── metadata/v1/metadata.proto
│   │   ├── storage/v1/storage.proto
│   │   ├── authz/v1/authz.proto
│   │   └── quota/v1/quota.proto
│   └── pkg/
│       ├── go/                # Generated Go stubs (*_pb.go, *_grpc.pb.go)
│       ├── py/                # Generated Python stubs (*_pb2.py, *_pb2_grpc.py)
│       ├── go-kit/            # Shared Go infrastructure
│       │   ├── middleware/    # metrics, rate limiter, circuit breaker
│       │   ├── tracing/       # OTel gRPC interceptor
│       │   ├── logger/
│       │   ├── tokens/        # JWT service
│       │   ├── client/db/     # PostgreSQL + tx manager
│       │   ├── kafka/         # producer + consumer
│       │   └── contextx/
│       ├── py_kit/            # Shared Python infrastructure
│       └── rust-kit/          # Shared Rust infrastructure
│
├── services/
│   ├── auth/                  # Go, JWT token issuance
│   ├── users/                 # Go, user CRUD + credentials
│   ├── authz/                 # Python, ReBAC engine
│   ├── metadata/              # Go, bucket + object metadata
│   ├── storage/               # Go, blob filesystem storage
│   └── quota/                 # Rust, per-user/bucket quota enforcement
│
├── infra/otel/                # Observability configs (Jaeger, Prometheus, Grafana, Kibana)
├── .github/workflows/         # CI per service (auth, users, metadata, storage, authz, quota)
└── docker-compose.yml         # Full stack (profiles: services, observability)
```

---

## Proto contracts

Proto files live in `shared/api/`. Generated stubs are committed to `shared/pkg/go/` and `shared/pkg/py/`.

| Service | package | RPC methods |
|---|---|---|
| Auth | `opens3.auth.v1` | `Login`, `GetAccessToken`, `GetRefreshToken`, `ValidateToken`, `HealthCheck` |
| Users | `opens3.user.v1` | `Create`, `Get`, `Delete`, `Update`, `UpdatePassword`, `ValidateCredentials`, `HealthCheck` |
| AuthZ | `opens3.authz.v1` | `Check`, `WriteTuple`, `DeleteTuple`, `Read`, `HealthCheck` |
| Metadata | `opens3.metadata.v1` | `CreateBucket`, `DeleteBucket`, `GetBucket`, `HeadBucket`, `ListBuckets`, `CreateObjectVersion`, `DeleteObjectMeta`, `GetObjectMeta`, `ListObjects`, `HealthCheck` |
| Storage | `opens3.storage.v1` | `StoreObject`, `RetrieveObject`, `DeleteObject`, `InitiateMultipartUpload`, `UploadPart`, `CompleteMultipartUpload`, `AbortMultipartUpload`, `HealthCheck` |
| Quota | `opens3.quota.v1` | `CheckQuota`, `UpdateUsage`, `SetQuota`, `SetLimit`, `HealthCheck` |

---

## Key domain concepts

### AuthZ: entity ID convention

All IDs in the AuthZ graph follow the format `prefix:name`:

| Prefix | Type | Example |
|---|---|---|
| `user:` | User | `user:alice` |
| `group:` | Group | `group:editors` |
| `bucket:` | Bucket | `bucket:photos` |
| `object:` | Object | `object:photos/cat.jpg` |
| `folder:` | Folder | `folder:reports/2024` |

### AuthZ: permission level hierarchy

```
read < write < create < delete < admin
```

A `HAS_PERMISSION` edge with `level: write` also grants `write`, `create`, `delete`, `admin`.

### AuthZ: Neo4j edge types

| Edge | From → To | Meaning |
|---|---|---|
| `MEMBER_OF` | User/Group → Group | Group membership (transitive) |
| `HAS_PERMISSION` | User/Group → Resource | Permission with level |
| `PARENT_OF` | Resource → Resource | Hierarchy (bucket → object) |
| `OWNER_OF` | User → Resource | Full access (legacy = admin) |

### AuthZ: Redis cache

Key: `auth_decision:{subject}:{action}:{object}`, TTL 30s.  
Invalidation: AuthZ publishes to `auth-changes` topic; `cache_invalidator` process consumes it.

### S3 → ReBAC mapping

| S3 operation | action | resource |
|---|---|---|
| `GetObject` | `read` | `object:{bucket}/{key}` |
| `PutObject` | `write` | `object:{bucket}` (**bucket**, not the object — it doesn't exist yet) |
| `DeleteObject` | `delete` | `object:{bucket}/{key}` |
| `ListBucket` | `read` | `bucket:{bucket}` |
| `CreateBucket` | — | no check needed — any user can create |
| `DeleteBucket` | `delete` | `bucket:{bucket}` |

### Metadata: entity hierarchy

```
Bucket
  └── Object (key, e.g. photos/2026/cat.jpg)
        └── Version (immutable blob_id + size + ETag)
              current_version_id → pointer to active version
```

### Quota: hot-path architecture

In-memory DashMap (~100ns lookup) → Redis persistence (flush every 1s).  
Reserve-and-rollback pattern: check user quota → check bucket quota → commit.

---

## Kafka topics

| Topic | Producer | Consumer | Event |
|---|---|---|---|
| `object-stored` | Storage | Metadata | blob fully written |
| `object-deleted` | Metadata | Storage, AuthZ | object marked deleted |
| `bucket-deleted` | Metadata | AuthZ | bucket deleted |
| `auth-changes` | AuthZ | AuthZ cache_invalidator | graph changed |
| `auth-audit` | AuthZ | — (log sink) | every Check decision |

---

## Full operation flows

### PutObject (planned, Gateway not implemented)

```
Client → PUT /{bucket}/{key}
  → Gateway: extract user_id from JWT
  → Check("user:{uid}", "write", "object:{bucket}") → AuthZ → ALLOW
  → CheckQuota(user_id, bucket) → Quota → ALLOW
  → StoreObject(stream) → Storage → { blob_id, checksum_md5 }
  → CreateObjectVersion(bucket, key, blob_id, size, etag) → Metadata → { version_id }
  → WriteTuple("bucket:{bucket}", "PARENT_OF", "object:{bucket}/{key}") → AuthZ
  → UpdateUsage(user_id, bucket, size) → Quota
  → 200 OK { ETag, version_id }
```

### GetObject

```
Client → GET /{bucket}/{key}
  → Check("user:{uid}", "read", "object:{bucket}/{key}") → AuthZ → ALLOW
  → GetObjectMeta(bucket, key) → Metadata → { blob_id, size, etag, content_type }
  → RetrieveObject(blob_id) → Storage → stream bytes
  → 200 OK + body stream
```

### CreateBucket

```
Client → PUT /{bucket}
  → [no AuthZ check]
  → CreateBucket(name, owner_id) → Metadata → { bucket_id }
  → WriteTuple("user:{uid}", "OWNER_OF", "bucket:{bucket}") → AuthZ
  → 200 OK
```

---

## gRPC → HTTP error mapping

| gRPC status | HTTP |
|---|---|
| `NOT_FOUND` | 404 |
| `ALREADY_EXISTS` | 409 |
| `FAILED_PRECONDITION` | 409 (e.g. bucket not empty) |
| `PERMISSION_DENIED` | 403 |
| `RESOURCE_EXHAUSTED` | 507 (quota exceeded) |
| `INVALID_ARGUMENT` | 400 |
| `UNAVAILABLE` | 503 |
| `INTERNAL` | 500 |

S3-compatible XML error format:
```xml
<Error>
  <Code>NoSuchBucket</Code>
  <Message>The specified bucket does not exist</Message>
  <Resource>/my-bucket</Resource>
  <RequestId>abc123</RequestId>
</Error>
```

---

## Environment variables

### Auth (`:50050`)
| Var | Default | Required |
|---|---|---|
| `GRPC_PORT` | `50050` | |
| `REDIS_HOST` | `localhost` | |
| `REDIS_PORT` | `6379` | |
| `JWT_SECRET` | — | ✅ |
| `JWT_ALGORITHM` | `HS256` | |
| `USERS_GRPC_ADDR` | `users:50054` | |

### Users (`:50054`)
| Var | Default | Required |
|---|---|---|
| `GRPC_PORT` | `50054` | |
| `DATABASE_URL` | — | ✅ PostgreSQL DSN |

### AuthZ (`:50051`)
| Var | Default | Required |
|---|---|---|
| `GRPC_PORT` | `50051` | |
| `NEO4J_URI` | `bolt://localhost:7687` | |
| `NEO4J_USER` | `neo4j` | |
| `NEO4J_PASSWORD` | — | ✅ |
| `REDIS_HOST` | `localhost` | |
| `REDIS_PORT` | `6379` | |
| `CACHE_TTL_SECONDS` | `30` | |
| `KAFKA_BOOTSTRAP` | `localhost:9092` | |

### Metadata (`:50052`)
| Var | Default | Required |
|---|---|---|
| `GRPC_PORT` | `50052` | |
| `DATABASE_URL` | — | ✅ PostgreSQL DSN |
| `DB_MAX_CONNECTIONS` | `20` | |
| `KAFKA_BOOTSTRAP` | `localhost:9092` | |

### Storage (`:50053`)
| Var | Default | Required |
|---|---|---|
| `GRPC_PORT` | `50053` | |
| `DATA_DIR` | `/data/blobs` | |
| `MULTIPART_DIR` | `/data/multipart` | |
| `MAX_CHUNK_SIZE_BYTES` | `8388608` | 8 MB |

### Quota (`:50055`)
| Var | Default | Required |
|---|---|---|
| `GRPC_PORT` | `50055` | |
| `REDIS_HOST` | `localhost` | |
| `REDIS_PORT` | `6379` | |
| `FLUSH_INTERVAL_MS` | `1000` | |

---

## Database schemas

### PostgreSQL — Metadata Service

```sql
CREATE TABLE buckets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT UNIQUE NOT NULL,
    owner_id   UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE objects (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket_id          UUID NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
    key                TEXT NOT NULL,
    current_version_id UUID NULL,
    UNIQUE (bucket_id, key)
);

CREATE TABLE versions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id    UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    blob_id      UUID NOT NULL,
    size_bytes   BIGINT NOT NULL,
    etag         TEXT NOT NULL,
    content_type TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted   BOOLEAN NOT NULL DEFAULT false
);
```

Indexes: `objects(bucket_id, key)`, `versions(object_id)`, `objects(bucket_id, key text_pattern_ops)` for prefix-based `ListObjects`.

---

## Local development

```bash
# Start all services + infra
docker compose --profile services up --build -d

# Start with observability (Jaeger, Prometheus, Grafana, Kibana)
docker compose --profile services --profile observability up --build -d

# Shortcuts via make
make up-services
make up-observability
make down
make down-volumes
make rebuild
```

### Observability UIs

| UI | Address | Credentials |
|---|---|---|
| Jaeger | http://localhost:16686 | — |
| Prometheus | http://localhost:9090 | — |
| Grafana | http://localhost:3000 | — |
| Kibana | http://localhost:5601 | — |
| Neo4j Browser | http://localhost:7474 | neo4j / password123 |

---

## Testing

Each service has unit tests and (where applicable) integration tests.

```bash
# Go services (from repo root via go.work)
go test ./services/auth/...
go test ./services/users/...
go test ./services/metadata/...
go test ./services/storage/...

# Python — AuthZ
cd services/authz
pytest tests/unit/
pytest tests/integration/  # requires Neo4j running

# Rust — Quota
cd services/quota
cargo test                         # unit + in-process gRPC tests
TEST_REDIS_URL=redis://localhost:6379 cargo test -- --include-ignored  # with Redis
```

CI runs per-service via `.github/workflows/` on push to `main` and on PRs.

---

## Roadmap

| Phase | Status | What |
|---|---|---|
| **Phase 0** | ✅ Done | Contracts, Docker Compose, shared infrastructure |
| **Phase 1** | 🔄 In Progress | MVP: PutObject + GetObject end-to-end |
| **Phase 2** | ⏳ | CreateBucket, DeleteBucket, DeleteObject, Kafka wiring, versioning |
| **Phase 3** | ⏳ | Multipart upload, object sharing, full S3 compatibility |
| **Phase 4** | ⏳ | Audit, E2E tests |

Each phase ends with a working demo.

---

## Service boundaries

| Service | Does NOT |
|---|---|
| **AuthZ** | authenticate users, store metadata, handle bytes, know about HTTP |
| **Metadata** | store bytes, check permissions, know about S3 API |
| **Storage** | check permissions, store metadata, know about object keys |
| **Quota** | enforce auth, store metadata, know about blobs |
| **Auth** | authorize (that's AuthZ), manage user profiles |
| **Users** | issue tokens, know about S3, check permissions |
| **Gateway** | store data, make authorization decisions, know about graph structure |

---

## Working guidelines for AI assistants

### Commits
- Commit **granularly** — one logical change per commit.
- No `Co-Authored-By` lines.
- Commit message format: `type(scope): short description` (e.g. `feat(metadata): add ListObjects handler`).

### Tests
- Every new feature or bug fix **must** include tests or an explicit offer to add them.
- Go: table-driven tests with `testify`, mocks via `minimock`.
- Python: `pytest`, fixtures in `conftest.py`.
- Rust: `#[cfg(test)]` modules for unit tests, `tests/` for integration.
- Integration tests that require external services (Neo4j, Redis) must be skippable without infra.

### Documentation hygiene
- When a gRPC contract changes (proto file, RPC name, field), update:
  - The relevant proto file in `shared/api/`
  - `README.md` (service section)
  - `CLAUDE.md` (proto contracts table, env vars if needed)
  - The service's own `README.md`
- When a new service is added, add it to the services table and architecture diagram in both `CLAUDE.md` and `README.md`.
- When env vars change, update both the service's own config and the env vars section here.

### Code style
- No comments explaining WHAT the code does — only WHY, when non-obvious.
- No speculative abstractions. Implement only what the current task requires.
- Error handling only at system boundaries (user input, external calls). Do not wrap internal errors defensively.
