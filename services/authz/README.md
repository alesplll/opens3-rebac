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
| `PARENT_OF` | Resource → Resource | Resource hierarchy |

**Level hierarchy:** `admin` ⊇ `delete` ⊇ `create` ⊇ `write` ⊇ `read`

### gRPC API

| RPC | Description |
|---|---|
| `Check(subject, action, object)` | Returns `{allowed: bool, reason: string}` |
| `WriteTuple(subject, relation, object, level?)` | Add a relationship |
| `DeleteTuple(subject, relation, object)` | Remove a relationship |
| `Read(subject)` | List all outgoing relationships |
| `HealthCheck` | Checks Neo4j + Redis connectivity |

---

## Project Structure — Every Folder, Every File

Полная карта проекта с объяснением что делает каждый файл и зачем он существует.

```
services/authz/
├── entrypoints/                      # Точки входа — отсюда запускаются процессы
│   ├── server/
│   │   ├── main.py                   # [1] Запускает gRPC сервер
│   │   └── servicer.py               # [2] gRPC хендлер (transport layer)
│   └── cache_invalidator.py          # [3] Kafka consumer (отдельный процесс)
│
├── internal/                         # Весь внутренний код
│   ├── config.py                     # [4] Конфигурация из env vars
│   ├── container.py                  # [5] Dependency injection
│   ├── types.py                      # [6] Доменный тип Tuple
│   │
│   ├── permission/                   # SERVICE слой — бизнес-логика
│   │   ├── interfaces.py             # [7] Протокол GraphStore
│   │   └── service.py                # [8] PermissionService — главный оркестратор
│   │
│   ├── repositories/                 # REPOSITORY слой — работа с данными
│   │   ├── neo4j/
│   │   │   ├── schema.py             # [9]  Схема графа Neo4j
│   │   │   └── store.py              # [10] Neo4jStore — запросы к графу
│   │   ├── cache/
│   │   │   ├── interfaces.py         # [11] ABC DecisionCache
│   │   │   ├── redis_cache.py        # [12] RedisDecisionCache — кэш решений
│   │   │   └── invalidation_consumer.py  # [13] CacheInvalidationConsumer
│   │   └── kafka/
│   │       └── producer.py           # [14] AuditProducer — отправка событий
│   │
│   └── metric/
│       └── authz_metrics.py          # [15] Метрики OTel
│
├── tests/
│   ├── unit/
│   │   └── test_permission_service.py  # [16] Юнит-тесты
│   └── integration/
│       └── test_neo4j_store.py         # [17] Интеграционные тесты 
│
└── proto/
    └── generate.sh                   # Скрипт регенерации gRPC стабов
```

### Подробно по каждому файлу

**[1] `entrypoints/server/main.py`**
Точка входа gRPC сервера. Запускается командой `python entrypoints/server/main.py`.
Делает ровно три вещи: инициализирует observability (logger → metrics → tracing), создаёт `Container` со всеми зависимостями, запускает gRPC сервер на порту 50051. Настраивает graceful shutdown по SIGINT/SIGTERM.

**[2] `entrypoints/server/servicer.py`**
Transport layer — gRPC хендлер. Переводит protobuf запросы в вызовы `PermissionService` и обратно.
Знает про protobuf, enum-ы, gRPC статусы. Не знает про Neo4j, Redis, Kafka — это не его дело.
Валидирует входные данные (нет action → `INVALID_ARGUMENT`). Вызывает только `container.rebac`.

**[3] `entrypoints/cache_invalidator.py`**
Точка входа второго процесса. Запускается отдельно от gRPC сервера.
Создаёт `RedisDecisionCache` и `CacheInvalidationConsumer`, запускает бесконечный цикл `consumer.run()`.
Существует отдельно потому что Kafka consumer — это вечный блокирующий цикл, который нельзя совмещать с gRPC сервером в одном процессе.

**[4] `internal/config.py`**
Единственное место где читаются env vars. Читает их один раз в `__init__`, хранит в приватных полях.
Реализует протоколы `py_kit`: `LoggerConfig`, `MetricsConfig`, `TracingConfig` — чтобы передавать `cfg` в `logger.init(cfg)`, `metric.init(cfg)` и т.д.
Создаёт глобальный синглтон `cfg` при импорте.

**[5] `internal/container.py`**
Dependency Injection контейнер. Единственное место где создаются реальные объекты инфраструктуры.
Читает конфиг, создаёт `Neo4jStore`, `RedisDecisionCache`, `AuditProducer`, `PermissionService` и соединяет их вместе.
`main.py` создаёт один `Container(cfg)` и передаёт его в `servicer.py`. Больше нигде `Neo4jStore()` не создаётся.

**[6] `internal/types.py`**
Единственный доменный тип — `Tuple`. Иммутабельный датакласс `(subject, relation, object, level?)`.
Используется везде: в сервисе, хранилище, аудите. Это "язык" которым общаются слои между собой.

**[7] `internal/permission/interfaces.py`**
Протокол `GraphStore` — контракт который должно выполнять любое хранилище графа.
Описывает методы: `write_tuple`, `read_tuples`, `delete_tuple`, `check`, `health`, `close`.
`PermissionService` зависит от этого протокола, а не от конкретного `Neo4jStore`. Это позволяет подменять реализацию в тестах через `MagicMock`.

**[8] `internal/permission/service.py`**
Главный файл сервиса. `PermissionService` — оркестратор который реализует бизнес-логику авторизации.
Знает порядок операций: сначала кэш → потом граф → потом кэшировать → потом аудит.
Записывает метрики (cache hit/miss, latency), создаёт OTel spans, пишет логи.
Методы: `check`, `write_tuple`, `delete_tuple`, `read_tuples`, `health_check`.

**[9] `internal/repositories/neo4j/schema.py`**
Константы и типы для работы с Neo4j графом.
`NodeLabel` — какие бывают узлы (User, Group, Resource…).
`RelationType` — какие бывают рёбра (MEMBER_OF, HAS_PERMISSION, PARENT_OF…).
`ALLOWED_LEVELS_PER_ACTION` — словарь "action → какие уровни разрешений его покрывают" (например, `write` покрывается уровнями write/create/delete/admin).
`infer_node_label()` — определяет тип узла по префиксу ID (`user:` → User, `bucket:` → Resource).

**[10] `internal/repositories/neo4j/store.py`**
`Neo4jStore` — реализация `GraphStore`. Выполняет Cypher-запросы к Neo4j.
`check()` — транзитивный обход `(subject)-[:MEMBER_OF*0..]->(x)-[:HAS_PERMISSION]->(object)` с проверкой уровня.
`write_tuple()` / `delete_tuple()` / `read_tuples()` — CRUD операции с рёбрами графа.
`health()` — вызывает `driver.verify_connectivity()`, бросает исключение если Neo4j недоступен.

**[11] `internal/repositories/cache/interfaces.py`**
ABC `DecisionCache` — контракт кэша.
Методы: `get(subject, action, object)` → `bool | None`, `set(...)` с TTL, `health()`.
Абстракция нужна чтобы в тестах подменять Redis на `MagicMock`.

**[12] `internal/repositories/cache/redis_cache.py`**
`RedisDecisionCache` — реализация `DecisionCache` поверх Redis.
Ключи в формате `auth_decision:{subject}:{action}:{object}`, значения `"1"` (allow) / `"0"` (deny).
`health()` — вызывает `_client.ping()`.

**[13] `internal/repositories/cache/invalidation_consumer.py`**
`CacheInvalidationConsumer` — Kafka consumer для инвалидации кэша.
Читает топик `auth-changes`. На события `tuple_written` / `tuple_removed` извлекает паттерны из `invalidation_hints` и делает Redis `SCAN` + `DELETE` по паттерну.
Например паттерн `auth_decision:user:alice:*:bucket:photos` удалит все кэшированные решения для alice на bucket:photos.

**[14] `internal/repositories/kafka/producer.py`**
`AuditProducer` — Kafka producer для аудит-событий.
`send_tuple_event()` — пишет в топик `auth-changes` событие о изменении графа (tuple_written / tuple_removed) вместе с `invalidation_hints` для cache invalidator.
`send_decision_event()` — пишет в топик `auth-audit` событие ACCESS_GRANTED / ACCESS_DENIED на каждый Check.

**[15] `internal/metric/authz_metrics.py`**
OTel/Prometheus метрики специфичные для authz.
`authz_cache_decisions_total{result=hit|miss}` — эффективность кэша.
`authz_decisions_total{action, result=allow|deny}` — статистика решений.
`authz_neo4j_query_duration_seconds{query_type}` — latency запросов к Neo4j.
Инициализируется один раз через `init()` после `metric.init_otel_metrics()`.

**[16] `tests/unit/test_permission_service.py`**
Юнит-тесты для `PermissionService` и `PermissionServiceServicer`.
Все внешние зависимости (store, cache, audit) заменены `MagicMock`.
Проверяет: логику кэша, аудит-события, делегирование в store, HealthCheck через `store.health()` / `cache.health()`.
Не требует запущенной инфраструктуры.

**[17] `tests/integration/test_neo4j_store.py`**
Интеграционные тесты для `Neo4jStore` — работают с реальным Neo4j.
Проверяют: транзитивный обход графа, иерархию уровней (write ⊇ read), запись/удаление рёбер, legacy-рёбра (OWNER_OF, VIEWER), S3 entity IDs.
Автоматически пропускаются если Neo4j недоступен.

---

## Import Map — кто кого импортирует

```
main.py
  → config.py                              (cfg — один раз при старте)
  → container.py                           (Container(cfg))
  → servicer.py                            (PermissionServiceServicer(container))
  → metric/authz_metrics.py               (init())
  → shared/py_kit                          (logger, metric, tracing)

servicer.py
  → container.py                           (тип аннотации Container)
  → types.py                               (Tuple)
  → permission/service.py                  (container.rebac — все вызовы идут сюда)
  → shared/py/authz/v1                     (protobuf классы)

container.py
  → config.py                              (Config)
  → repositories/neo4j/store.py            (Neo4jStore)
  → repositories/cache/redis_cache.py      (RedisDecisionCache)
  → repositories/kafka/producer.py         (AuditProducer)
  → permission/service.py                  (PermissionService)

permission/service.py
  → permission/interfaces.py               (GraphStore — тип store)
  → repositories/cache/interfaces.py       (DecisionCache — тип cache)
  → repositories/kafka/producer.py         (AuditProducer — тип audit_producer)
  → types.py                               (Tuple)
  → metric/authz_metrics.py               (record_cache_hit/miss, record_decision, record_neo4j_query)
  → shared/py_kit                          (logger, start_span)

repositories/neo4j/store.py
  → repositories/neo4j/schema.py           (NodeLabel, RelationType, ALLOWED_LEVELS_PER_ACTION, …)
  → types.py                               (Tuple)
  → shared/py_kit                          (logger)

repositories/cache/redis_cache.py
  → repositories/cache/interfaces.py       (DecisionCache — наследует)

repositories/cache/invalidation_consumer.py
  → repositories/cache/redis_cache.py      (RedisDecisionCache — для SCAN+DEL)

repositories/kafka/producer.py
  → types.py                               (Tuple)

cache_invalidator.py
  → repositories/cache/redis_cache.py      (RedisDecisionCache)
  → repositories/cache/invalidation_consumer.py  (CacheInvalidationConsumer)
```

### Частота вызовов в runtime (на один Check запрос)

| Вызов | Кол-во раз |
|---|---|
| `servicer.Check()` → `service.check()` | 1 |
| `service.check()` → `cache.get()` | 1 |
| `service.check()` → `store.check()` | 0 (cache hit) или 1 (cache miss) |
| `service.check()` → `cache.set()` | 0 (cache hit) или 1 (cache miss) |
| `service.check()` → `audit.send_decision_event()` | 1 (всегда) |
| `metric.record_*()` | 3–4 (cache + neo4j + decision) |

---

## Quick Start (local dev)

### Prerequisites

- Python 3.12
- Docker & Docker Compose
- [`grpcurl`](https://github.com/fullstorydev/grpcurl) for manual testing

### 1. Setup

```bash
cd rebac-auth-service

python -m venv venv
source venv/bin/activate
pip install -e ".[test]"

bash proto/generate.sh
```

### 2. Start infrastructure (Neo4j, Redis, Kafka)

From the **repository root:**

```bash
# All services + infra (recommended)
make up-services

# Or just the infra + authz only
docker compose up neo4j redis kafka zookeeper -d
docker compose --profile services up authz -d
```

- **Neo4j Browser:** http://localhost:7474 — login `neo4j` / `password123`
- **Redis:** `redis-cli ping` → `PONG`
- **Kafka:** `localhost:9092`

### 3. Run the gRPC server (local, without Docker)

```bash
source venv/bin/activate
python entrypoints/server/main.py
# → ReBAC Auth Service gRPC :50051
```

### 4. (Optional) Run cache invalidator

In a separate terminal:

```bash
source venv/bin/activate
python entrypoints/cache_invalidator.py
```

### Environment variables

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
docker build -t rebac-auth-service .

docker run --rm --network host rebac-auth-service
```

### Full stack (infra + service)

```bash
docker compose -f deploy/local/docker-compose.yml --profile app up -d
```

### Stop everything

From the repo root:

```bash
make down            # остановить все контейнеры
make down-volumes    # остановить + удалить данные (Neo4j, Redis)
```

### Rebuild after code changes

```bash
make rebuild         # пересобрать без кэша и перезапустить
```

---

## Manual Testing (grpcurl)

All commands below assume the server is running on `localhost:50051`.

### Verify service is up

```bash
grpcurl -plaintext localhost:50051 list
# → opens3.authz.v1.PermissionService
# → grpc.health.v1.Health
```

### Health check

```bash
# Standard gRPC health protocol (used by K8s probes)
grpcurl -plaintext -d '{"service": "opens3.authz.v1.PermissionService"}' \
  localhost:50051 grpc.health.v1.Health/Check
# → {"status": "SERVING"}

# Custom HealthCheck (checks Neo4j + Redis connectivity)
grpcurl -plaintext localhost:50051 opens3.authz.v1.PermissionService/HealthCheck
# → {"status": "SERVING"}
```

### Write relationships

```bash
# alex is a member of devops
grpcurl -plaintext -d '{"subject":"user:alex","relation":"RELATION_MEMBER_OF","object":"group:devops"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple

# devops has admin permission on server-1
grpcurl -plaintext -d '{"subject":"group:devops","relation":"RELATION_HAS_PERMISSION","object":"resource:server-1","level":"PERMISSION_LEVEL_ADMIN"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple

# viewers group has read-only on doc1
grpcurl -plaintext -d '{"subject":"group:viewers","relation":"RELATION_HAS_PERMISSION","object":"resource:doc1","level":"PERMISSION_LEVEL_READ"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple

# bob is a viewer
grpcurl -plaintext -d '{"subject":"user:bob","relation":"RELATION_MEMBER_OF","object":"group:viewers"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple

# Transitive chain: alice → payments → finance → billing (read)
grpcurl -plaintext -d '{"subject":"user:alice","relation":"RELATION_MEMBER_OF","object":"group:payments"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple
grpcurl -plaintext -d '{"subject":"group:payments","relation":"RELATION_MEMBER_OF","object":"group:finance"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple
grpcurl -plaintext -d '{"subject":"group:finance","relation":"RELATION_HAS_PERMISSION","object":"resource:billing","level":"PERMISSION_LEVEL_READ"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple
```

### Check permissions

```bash
# alex: admin implies read on server-1 → ALLOW
grpcurl -plaintext -d '{"subject":"user:alex","action":"ACTION_ADMIN","object":"resource:server-1"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check

grpcurl -plaintext -d '{"subject":"user:alex","action":"ACTION_READ","object":"resource:server-1"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check

# eve has no rights → DENY
grpcurl -plaintext -d '{"subject":"user:eve","action":"ACTION_READ","object":"resource:server-1"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check

# bob: read allowed, write denied
grpcurl -plaintext -d '{"subject":"user:bob","action":"ACTION_READ","object":"resource:doc1"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check
grpcurl -plaintext -d '{"subject":"user:bob","action":"ACTION_WRITE","object":"resource:doc1"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check

# alice: transitive chain alice→payments→finance → read on billing, but not write
grpcurl -plaintext -d '{"subject":"user:alice","action":"ACTION_READ","object":"resource:billing"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check
grpcurl -plaintext -d '{"subject":"user:alice","action":"ACTION_WRITE","object":"resource:billing"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check
```

### Delete relationships

```bash
# Remove alex from devops → loses access to server-1
grpcurl -plaintext -d '{"subject":"user:alex","relation":"RELATION_MEMBER_OF","object":"group:devops"}' \
  localhost:50051 opens3.authz.v1.PermissionService/DeleteTuple
# → {"success": true}

# Verify access is gone
grpcurl -plaintext -d '{"subject":"user:alex","action":"ACTION_READ","object":"resource:server-1"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check
# → {"allowed": false}

# Deleting a non-existent tuple returns false
grpcurl -plaintext -d '{"subject":"user:nobody","relation":"RELATION_MEMBER_OF","object":"group:nobody"}' \
  localhost:50051 opens3.authz.v1.PermissionService/DeleteTuple
# → {"success": false}
```

### Read relationships

```bash
grpcurl -plaintext -d '{"subject":"user:alex"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Read

grpcurl -plaintext -d '{"subject":"group:devops"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Read
```

### Kafka audit events

```bash
# Список топиков
docker exec opens3-rebac-kafka-1 kafka-topics --bootstrap-server localhost:9092 --list

# Читать аудит-лог (каждый Check пишет туда)
docker exec opens3-rebac-kafka-1 kafka-console-consumer \
  --bootstrap-server localhost:9092 --topic auth-audit --from-beginning

# Инвалидация кэша (WriteTuple / DeleteTuple)
docker exec opens3-rebac-kafka-1 kafka-console-consumer \
  --bootstrap-server localhost:9092 --topic auth-changes --from-beginning
```

Events emitted:
- `tuple_written` / `tuple_removed` — on WriteTuple / DeleteTuple (includes `invalidation_hints` for Redis)
- `ACCESS_GRANTED` / `ACCESS_DENIED` — on every Check

### Redis cache

```bash
docker exec -it opens3-rebac-redis-1 redis-cli

# Внутри redis-cli:
KEYS *                                              # все ключи
KEYS auth_decision:*                               # только кэш решений
GET auth_decision:user:alex:read:resource:server-1 # 1 = allow, 0 = deny
TTL auth_decision:user:alex:read:resource:server-1 # оставшееся время жизни (default 30s)
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
venv/bin/python -m pytest tests/unit -v

# Unit tests with coverage
venv/bin/python -m pytest tests/unit -v --cov=internal --cov=entrypoints --cov-report=term-missing

# Integration tests (Neo4j must be running)
venv/bin/python -m pytest tests/integration -v -m integration
```

Integration tests auto-skip if Neo4j is unreachable. Run Neo4j first (from repo root):

```bash
docker compose up neo4j -d
```

---

## Proto regeneration

The source of truth is `shared/api/authz/v1/authz.proto` in the repo root.
Stubs are pre-generated and committed — no need to regenerate unless the proto changes.

After modifying the shared proto:

```bash
cd services/authz
bash proto/generate.sh
# Regenerates shared/pkg/py/authz/v1/authz_pb2.py and authz_pb2_grpc.py
```
