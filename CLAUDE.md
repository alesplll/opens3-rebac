# CLAUDE.md — opens3-rebac

> Этот файл помогает AI-ассистентам (Claude, Copilot и т.д.) быстро понять контекст проекта и давать точные, согласованные с архитектурой ответы.

---

## Что такое этот проект

**OpenS3** — распределённое объектное хранилище, совместимое с S3 API, с авторизацией на основе **ReBAC** (Relationship-Based Access Control). Учебный командный проект на 4 человека.

Репозиторий: https://github.com/alesplll/opens3-rebac  
Wiki: https://github.com/alesplll/opens3-rebac/wiki

---

## Команда и сервисы

| Сервис | Язык | Порт | Кто |
|---|---|---|---|
| **Gateway** | Go | `:8080` (HTTP) | Макс |
| **AuthZ (ReBAC)** | Python | `:50051` (gRPC) | Алекса (я) |
| **Metadata Service** | Python | `:50052` (gRPC) | Аня |
| **Data Node** | Go | `:50053` (gRPC) | Илья |

Связь между сервисами — **gRPC**. Клиент → Gateway — **HTTP/S3 API**.

---

## Архитектура одной строкой

```
Client (HTTP S3) → Gateway → [AuthZ (ReBAC) | Metadata Service | Data Node]
                                     ↕                  ↕
                                   Neo4j            PostgreSQL
                                   Redis
                                                Kafka (async events)
```

Gateway — единственная точка входа. Он **не хранит состояние**, только оркестрирует.

---

## Стек технологий

- **Go 1.23** — Gateway, Data Node
- **Python 3.12** — AuthZ, Metadata Service
- **gRPC / Protobuf** — внутренняя связь сервисов
- **PostgreSQL** — метаданные (buckets, objects, versions)
- **Neo4j** — граф прав доступа (ReBAC)
- **Redis** — кэш решений авторизации (TTL 30s)
- **Kafka** — асинхронные события между сервисами
- **Docker Compose** — локальная среда разработки
- **Prometheus + Grafana** — мониторинг (Phase 4)

---

## Протоколы / пакеты gRPC

| Сервис | proto package | service |
|---|---|---|
| AuthZ | `opens3.authz.v1` | `PermissionService` |
| Metadata | `opens3.metadata.v1` | `MetadataService` |
| Data Node | `opens3.storage.v1` | `DataStorageService` |

Proto-файлы: `proto/authz/v1/authz.proto`, `proto/metadata/v1/metadata.proto`, `proto/storage/v1/storage.proto`

> **Важно:** `services/authz/proto/authz.proto` (standalone) использует пакет `rebac.authz.v1`.
> До интеграции Gateway ↔ AuthZ его нужно синхронизировать с shared proto (`opens3.authz.v1`).
> См. `services/authz/issues/proto_package_sync.md`.

---

## Ключевые концепции

### Соглашение об ID сущностей (AuthZ)

Все ID в AuthZ имеют формат `prefix:name`:

| Префикс | Тип | Пример |
|---|---|---|
| `user:` | Пользователь | `user:alice` |
| `group:` | Группа | `group:editors` |
| `bucket:` | Бакет | `bucket:photos` |
| `object:` | Объект | `object:photos/cat.jpg` |
| `folder:` | Папка | `folder:reports/2024` |

### Маппинг S3 → ReBAC Check

| S3 операция | action | object |
|---|---|---|
| GetObject | `read` | `object:{bucket}/{key}` |
| PutObject | `write` | `object:{bucket}` (**бакет**, не объект) |
| DeleteObject | `delete` | `object:{bucket}/{key}` |
| ListBucket | `read` | `bucket:{bucket}` |
| CreateBucket | — | **Check не нужен**, любой пользователь может создать |
| DeleteBucket | `delete` | `bucket:{bucket}` |

> **Важно:** при `PutObject` Check делается на `object:{bucket}` (родительский бакет), потому что объект ещё не существует. После успешного сохранения Gateway вызывает `WriteTuple` чтобы добавить объект в граф.

### Иерархия уровней в ReBAC

```
read < write < create < delete < admin
```

Ребро типа `HAS_PERMISSION` с `level: write` разрешает также `write`, `create`, `delete`, `admin`. `admin` разрешает всё.

### Типы рёбер в Neo4j

| Ребро | От → К | Смысл |
|---|---|---|
| `MEMBER_OF` | User/Group → Group | Членство в группе (транзитивное) |
| `HAS_PERMISSION` | User/Group → Resource | Разрешение с уровнем |
| `PARENT_OF` | Resource → Resource | Иерархия (bucket → object) |
| `OWNER_OF` | User → Resource | Полный доступ (legacy = admin) |

### Иерархия сущностей Metadata

```
Bucket
  └── Object (ключ, например photos/2026/cat.jpg)
        └── Version (неизменяемый blob_id + size + ETag)
              current_version_id → указатель на актуальную версию
```

### Kafka топики

| Топик | Кто пишет | Кто читает | Событие |
|---|---|---|---|
| `object-stored` | Data Node | Metadata Service | blob полностью записан |
| `object-deleted` | Metadata Service | Data Node, AuthZ | объект помечен удалённым |
| `bucket-deleted` | Metadata Service | AuthZ | бакет удалён |
| `auth-changes` | AuthZ | AuthZ (cache_invalidator) | граф прав изменился |
| `auth-audit` | AuthZ | — (log sink) | каждый Check |

---

## Полные flow операций

### PutObject

```
Client → PUT /{bucket}/{key}
  → Gateway: извлечь user_id из JWT
  → Check("user:{uid}", "write", "object:{bucket}") → ReBAC → ALLOW
  → StoreObject(stream) → Data Node → { blob_id, checksum_md5 }
  → CreateObjectVersion(bucket, key, blob_id, size, etag) → Metadata → { version_id }
  → WriteTuple("bucket:{bucket}", "PARENT_OF", "object:{bucket}/{key}") → ReBAC
  → 200 OK { ETag, version_id }
```

### GetObject

```
Client → GET /{bucket}/{key}
  → Check("user:{uid}", "read", "object:{bucket}/{key}") → ReBAC → ALLOW
  → GetObjectMeta(bucket, key) → Metadata → { blob_id, size, etag, content_type }
  → RetrieveObject(blob_id) → Data Node → stream bytes
  → 200 OK + body stream
```

### DeleteObject

```
Client → DELETE /{bucket}/{key}
  → Check("user:{uid}", "delete", "object:{bucket}/{key}") → ReBAC → ALLOW
  → DeleteObjectMeta(bucket, key) → Metadata
  → Metadata → Kafka: ObjectDeleted { blob_id }
  → 204 No Content
  (асинхронно: Data Node удаляет файл, AuthZ удаляет узел из графа)
```

### CreateBucket

```
Client → PUT /{bucket}
  → [нет Check]
  → CreateBucket(name, owner_id) → Metadata → { bucket_id }
  → WriteTuple("user:{uid}", "OWNER_OF", "bucket:{bucket}") → ReBAC
  → 200 OK
```

---

## Обработка ошибок: gRPC → HTTP

| gRPC | HTTP |
|---|---|
| `NOT_FOUND` | 404 |
| `ALREADY_EXISTS` | 409 |
| `FAILED_PRECONDITION` | 409 (например, бакет не пуст) |
| `PERMISSION_DENIED` / allowed=false | 403 |
| `INVALID_ARGUMENT` | 400 |
| `RESOURCE_EXHAUSTED` | 507 |
| `UNAVAILABLE` | 503 |
| `INTERNAL` | 500 |

Ошибки возвращаются клиенту в S3-совместимом XML:

```xml
<Error>
  <Code>NoSuchBucket</Code>
  <Message>The specified bucket does not exist</Message>
  <Resource>/my-bucket</Resource>
  <RequestId>abc123</RequestId>
</Error>
```

---

## Конфигурация (env vars по сервисам)

### AuthZ

| Var | Default | Описание |
|---|---|---|
| `GRPC_PORT` | `50051` | gRPC порт |
| `NEO4J_URI` | `bolt://localhost:7687` | Neo4j |
| `NEO4J_USER` | `neo4j` | |
| `NEO4J_PASSWORD` | — | **обязателен** |
| `REDIS_HOST` | `localhost` | |
| `REDIS_PORT` | `6379` | |
| `CACHE_TTL_SECONDS` | `30` | TTL кэша решений |
| `KAFKA_BOOTSTRAP` | `localhost:9092` | |
| `KAFKA_CHANGES_TOPIC` | `auth-changes` | |
| `KAFKA_AUDIT_TOPIC` | `auth-audit` | |

### Metadata Service

| Var | Default | Описание |
|---|---|---|
| `GRPC_PORT` | `50052` | |
| `DATABASE_URL` | — | **обязателен** PostgreSQL DSN |
| `DB_MAX_CONNECTIONS` | `20` | |
| `KAFKA_BOOTSTRAP` | `localhost:9092` | |
| `KAFKA_OBJECT_DELETED_TOPIC` | `object-deleted` | |
| `KAFKA_OBJECT_STORED_TOPIC` | `object-stored` | |

### Data Node

| Var | Default | Описание |
|---|---|---|
| `GRPC_PORT` | `50053` | |
| `DATA_DIR` | `/data/blobs` | директория blob-файлов |
| `MULTIPART_DIR` | `/data/multipart` | временные part-файлы |
| `KAFKA_BOOTSTRAP` | `localhost:9092` | |
| `KAFKA_OBJECT_DELETED_TOPIC` | `object-deleted` | |
| `KAFKA_OBJECT_STORED_TOPIC` | `object-stored` | |
| `MAX_CHUNK_SIZE_BYTES` | `8388608` | 8 MB |

### Gateway

| Var | Default | Описание |
|---|---|---|
| `HTTP_PORT` | `8080` | |
| `AUTHZ_GRPC_ADDR` | `authz:50051` | |
| `METADATA_GRPC_ADDR` | `metadata:50052` | |
| `STORAGE_GRPC_ADDR` | `storage:50053` | |
| `JWT_SECRET` | — | **обязателен** |
| `JWT_ALGORITHM` | `HS256` | |
| `MAX_UPLOAD_SIZE_BYTES` | `5368709120` | 5 GB |
| `GRPC_TIMEOUT_MS` | `5000` | |

---

## Схема БД (PostgreSQL — Metadata Service)

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

Индексы: `objects(bucket_id, key)`, `versions(object_id)`, `objects(bucket_id, key text_pattern_ops)` для ListObjects с prefix.

---

## Структура файловой системы Data Node

```
/data/blobs/
    {blob_id}            ← финальные объекты (UUID v4 имя файла)
    ...
/data/multipart/
    {upload_id}/
        part_1
        part_2
        ...
```

> Рекомендация: вложенность `blobs/{blob_id[0:2]}/{blob_id}` чтобы не класть >100k файлов в одну директорию.

---

## Health checks

- **AuthZ / Metadata / Data Node**: `rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse)` — определён в каждом proto-файле. `ServingStatus`: `SERVING` | `NOT_SERVING` | `UNKNOWN`.
  - AuthZ: `NOT_SERVING` если Neo4j или Redis недоступны
  - Metadata: `NOT_SERVING` если PostgreSQL или Kafka недоступны
  - Data Node: `NOT_SERVING` если `DATA_DIR` недоступна
- **Gateway**: `GET /health` → liveness, `GET /ready` → readiness (вызывает `HealthCheck` у всех трёх gRPC-сервисов)

---

## Roadmap

| Фаза | Статус | Что |
|---|---|---|
| **Phase 0** | ✅ Done | Синхронизация, контракты, Docker Compose |
| **Phase 1** | 🔄 In Progress | MVP: PutObject + GetObject end-to-end |
| **Phase 2** | ⏳ | CreateBucket, DeleteBucket, DeleteObject, Kafka, права доступа, версионирование |
| **Phase 3** | ⏳ | Multipart upload, шеринг объектов, S3-совместимость |
| **Phase 4** | ⏳ | Аудит, мониторинг, E2E тесты |

**Принцип:** каждая фаза заканчивается рабочим демо, а не просто написанным кодом.

---

## Частые вопросы / ловушки

**Q: Почему при PutObject Check делается на бакет, а не объект?**  
A: Объект ещё не существует — нечего проверять. Разрешение на запись в бакет означает возможность создавать объекты внутри. После создания Gateway добавляет `PARENT_OF` ребро в граф.

**Q: Кто генерирует blob_id?**  
A: Только Data Node. Metadata Service получает blob_id от Gateway, который получил его от Data Node.

**Q: Как работает кэш AuthZ?**  
A: Redis, ключ `auth_decision:{subject}:{action}:{object}`, TTL 30s. При изменении графа AuthZ публикует в `auth-changes`, фоновый процесс `cache_invalidator` инвалидирует ключи по паттерну.

**Q: Data Node общается с Metadata Service напрямую?**  
A: Нет. Только через Kafka. Data Node → `object-stored` → Metadata. Metadata → `object-deleted` → Data Node.

**Q: Что значит `OWNER_OF` vs `HAS_PERMISSION{level: admin}`?**  
A: `OWNER_OF` — legacy ребро, полный доступ. В новом коде лучше использовать `HAS_PERMISSION` с `level: admin`. Граф-запросы поддерживают оба варианта через fallback.

**Q: Аутентификация vs авторизация?**  
A: Gateway аутентифицирует (JWT → user_id). AuthZ авторизует (user_id → ALLOW/DENY). AuthZ ничего не знает о JWT.

---

## Что НЕ делает каждый сервис (границы)

| Сервис | НЕ делает |
|---|---|
| **AuthZ** | аутентификацию, хранение метаданных, работу с байтами, знание об HTTP |
| **Metadata** | хранение байтов, проверку прав, знание об S3 API |
| **Data Node** | проверку прав, хранение метаданных, знание о ключах объектов |
| **Gateway** | хранение данных, принятие решений о правах, знание о структуре графа |
