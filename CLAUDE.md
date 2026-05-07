# CLAUDE.md — opens3-rebac

> Context for AI assistants. Contains only what can't be derived from reading the code.

---

## What this project is

**OpenS3-ReBAC** — distributed S3-compatible object storage with **ReBAC** (Relationship-Based Access Control) authorization. Team learning project (4 people).

Repo: https://github.com/alesplll/opens3-rebac

---

## Services

| Service | Lang | Port | Owner | Status |
|---|---|---|---|---|
| **Gateway** | Go | `:8080` HTTP | Max | ⚠️ Not implemented |
| **Auth** | Go | `:50050` gRPC | — | ✅ Complete |
| **AuthZ (ReBAC)** | Python | `:50051` gRPC | Alexa | ✅ Complete |
| **Metadata** | Go | `:50052` gRPC | Anya | ✅ Complete |
| **Storage** | Go | `:50053` gRPC | Ilya | ✅ Complete |
| **Users** | Go | `:50054` gRPC | — | ✅ Complete |
| **Quota** | Rust | `:50055` gRPC | — | ✅ Complete |

Proto source: `shared/api/`. Generated stubs: `shared/pkg/go/`, `shared/pkg/py/`.

---

## Architecture

```
Client (HTTP / S3 API)
        │
        ▼
   Gateway :8080       ← single entry point (NOT YET IMPLEMENTED)
  /   |    |    \
Auth AuthZ Meta Storage
  │    │    │      │
  │  Neo4j PostgreSQL filesystem
  │  Redis  Kafka
Users :50054 ── PostgreSQL
Quota :50055 ── Redis ── Kafka
```

---

## Key domain concepts

### AuthZ: entity ID format

All IDs in the AuthZ graph: `prefix:name` — e.g. `user:alice`, `bucket:photos`, `object:photos/cat.jpg`.

### AuthZ: permission hierarchy

```
read < write < create < delete < admin
```

A `HAS_PERMISSION` edge with `level: write` also grants `create`, `delete`, `admin`.

### AuthZ: Neo4j edge types

| Edge | Meaning |
|---|---|
| `MEMBER_OF` | Group membership (transitive) |
| `HAS_PERMISSION` | Permission with level |
| `PARENT_OF` | Resource hierarchy (bucket → object) |
| `OWNER_OF` | Full access (legacy = admin) |

### AuthZ: Redis cache

Key: `auth_decision:{subject}:{action}:{object}`, TTL 30s.  
Invalidation: AuthZ publishes to `auth-changes`; `cache_invalidator` process consumes it.

### S3 → ReBAC mapping

| S3 operation | action | resource |
|---|---|---|
| `GetObject` | `read` | `object:{bucket}/{key}` |
| `PutObject` | `write` | `object:{bucket}` ← **bucket**, not object (doesn't exist yet) |
| `DeleteObject` | `delete` | `object:{bucket}/{key}` |
| `ListBucket` | `read` | `bucket:{bucket}` |
| `CreateBucket` | — | no check — any user can create |
| `DeleteBucket` | `delete` | `bucket:{bucket}` |

### Quota: hot-path architecture

In-memory DashMap (~100ns) → Redis persistence (flush every 1s).  
Reserve-and-rollback: check user limit → check bucket limit → commit.

### Kafka topics

| Topic | Producer | Consumer |
|---|---|---|
| `object-stored` | Storage | Metadata |
| `object-deleted` | Metadata | Storage, AuthZ |
| `bucket-deleted` | Metadata | AuthZ |
| `auth-changes` | AuthZ | AuthZ cache_invalidator |
| `auth-audit` | AuthZ | — (log sink) |

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

## Roadmap

| Phase | Status | What |
|---|---|---|
| **Phase 0** | ✅ Done | Contracts, Docker Compose, shared infrastructure |
| **Phase 1** | 🔄 In Progress | MVP: PutObject + GetObject end-to-end |
| **Phase 2** | ⏳ | CreateBucket, DeleteBucket, DeleteObject, Kafka wiring, versioning |
| **Phase 3** | ⏳ | Multipart upload, object sharing, full S3 compatibility |
| **Phase 4** | ⏳ | Audit, E2E tests |

---

## Working guidelines

### Commits
- One logical change per commit. No `Co-Authored-By` lines.
- Format: `type(scope): short description` — e.g. `feat(metadata): add ListObjects handler`.

### Tests
- Every feature or bug fix must include tests or an explicit offer to add them.
- Go: table-driven with `testify`, mocks via `minimock`.
- Python: `pytest`, fixtures in `conftest.py`.
- Rust: `#[cfg(test)]` for unit, `tests/` for integration.
- Integration tests requiring external infra (Neo4j, Redis) must be skippable without it.

### Documentation hygiene
When a gRPC contract changes or a new service is added, update:
- `shared/api/` proto file
- `README.md` (service section + architecture diagram)
- `CLAUDE.md` (services table, domain concepts if affected)
- The service's own `README.md`

### Follow existing patterns
Before implementing anything foundational (handler, service, repository, config, migrations, tests), **look at how it's done in the project first**.

The canonical reference is **`services/users/`** — it covers handler structure, service layer with interface + mock, PostgreSQL repository, domain models, env config, table-driven tests, and SQL migration layout.

### Code style
- No comments explaining WHAT — only WHY when non-obvious.
- No speculative abstractions. Implement only what the task requires.
- Error handling only at system boundaries (user input, external calls).
