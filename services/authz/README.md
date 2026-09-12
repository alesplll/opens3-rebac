# AuthZ Service

Python 3.12 gRPC-сервис Relationship-Based Access Control. Хранит graph в Neo4j,
кэширует решения в Redis и публикует audit/tuple events в Kafka.

## Назначение и возможности

Проверка доступа, запись/удаление отношений, чтение прямых связей и диагностика зависимостей. Аутентификацию клиента и авторизацию инициатора изменения прав должен выполнять вызывающий компонент.

## Архитектура и зависимости

```text
gRPC PermissionService
        ↓
PermissionServiceServicer
        ├── Redis decision cache
        ├── Neo4j relationship store
        └── Kafka producer

отдельный process: Kafka invalidation consumer → Redis cache
```

Neo4j является источником отношений. Redis ускоряет `Check`, Kafka переносит
audit/изменения и может запускать best-effort invalidation. Server и consumer —
разные entrypoints и разворачиваются отдельно.

## Структура проекта

```text
entrypoints/server/main.py                 gRPC process
entrypoints/server/servicer.py             protobuf transport
entrypoints/cache_invalidator.py           cache invalidation process
internal/permission/service.py             application logic
internal/repositories/neo4j/               graph queries
internal/repositories/cache/               Redis cache и consumer
internal/repositories/kafka/               producer
internal/config.py                          environment configuration
tests/unit/                                 isolated tests
tests/integration/                          Neo4j-backed tests
```

Generated Python protobuf находится в `shared/pkg/py`; его не редактируют вручную.

## API

Source of truth: `shared/api/authz/v1/authz.proto`, package
`opens3.authz.v1`, service `PermissionService`.

| RPC | Поведение |
|---|---|
| `Check` | cache lookup → graph lookup при miss → cache write → audit event |
| `WriteTuple` | добавить relation в Neo4j и опубликовать tuple event |
| `DeleteTuple` | удалить точную relation и опубликовать event при успехе |
| `Read` | вернуть прямые исходящие relations subject |
| `HealthCheck` | custom проверка Neo4j/Redis |

Стандартный `grpc.health.v1.Health` также зарегистрирован, но получает SERVING при
старте процесса и не вызывает зависимости для каждого probe.

### Как выполняется Check

1. Servicer валидирует protobuf enum и формирует cache key.
2. При cache hit возвращается сохранённое решение.
3. При miss Neo4j проверяет прямые и group-derived permissions.
4. Решение записывается в Redis с TTL.
5. Audit event отправляется через Kafka producer.

Публикация audit event не входит в атомарную транзакцию graph read. Ошибка callback доставки Kafka логируется. Синхронное исключение из `produce()`
не перехватывается в Check и может прервать RPC; гарантии независимости ответа от
ошибок аудита пока нет.

## Данные и основные сценарии

### Модель отношений

Публичный protobuf принимает:

- `MEMBER_OF`: user/group → group;
- `HAS_PERMISSION`: user/group → bucket/object с permission level;
- `PARENT_OF`: resource → resource.

Уровни: `admin > delete > create > write > read`; уровень разрешает запрошенное
действие того же или более низкого уровня. Для владельца в публичном API нужно
использовать `HAS_PERMISSION` + `ADMIN`. `OWNER_OF` читается store только как
legacy relation и отсутствует в публичном enum.

Текущий `Check` следует по цепочке `MEMBER_OF` и ищет `HAS_PERMISSION` на точно
запрошенном ресурсе. Само наличие `PARENT_OF` не даёт наследование прав
bucket → object. Такое наследование требует отдельной политики и тестов.

Entity IDs используются в формате:

```text
user:<uuid>
group:<name>
bucket:<name>
object:<bucket>/<key>
```

Servicer валидирует protobuf enums. Полная проверка всех строковых prefixes пока
не является гарантией API.

Пример цепочки:

```text
(user:alex)-[:MEMBER_OF]->(group:devops)
(group:devops)-[:HAS_PERMISSION {level: "admin"}]->(bucket:logs)
```

Она разрешает действия на `bucket:logs` согласно hierarchy уровней. Она не
разрешает автоматически действие на `object:logs/app.log`, пока в query явно не
реализована семантика `PARENT_OF`.

### Кэш и отзыв доступа

Решения сохраняются в Redis на 30 секунд. Основной Docker entrypoint запускает
только gRPC server; `entrypoints/cache_invalidator.py` нужно разворачивать
отдельно, если используется Kafka invalidation.

Даже с consumer текущая инвалидация best-effort. Удаление group membership не
перечисляет все resource decisions, полученные через группу, а конкурентный
graph-check способен записать старый ALLOW после очистки. Поэтому нельзя обещать
немедленный отзыв: TTL ограничивает срок жизни отдельной записи с момента её сохранения,
но не задаёт строгие 30 секунд с момента отзыва: старый graph-check может
закончиться позже. Для SLA нужны dependency-aware invalidation либо versioned epochs.

Не увеличивайте/sliding-extend ALLOW TTL без независимого hard deadline: потеря
invalidation тогда может продлевать уже отозванный доступ неограниченно.

Безопасные варианты развития: dependency-aware invalidation всех зависимых
решений, epochs/generations в cache key либо короткий bounded TTL как временная
граница stale access. Выбор и race tests зафиксированы в issue #67.

### Kafka

Текущая конфигурация использует один producer с topic `auth-changes` по умолчанию.
Туда отправляются tuple events и authorization decisions. Payload решения содержит
`event_type`, `subject`, `action`, `object` и `timestamp` (миллисекунды); он не содержит
документированные ранее `from_cache`, `level` или `timestamp_ms`; отдельного
`allowed` тоже нет — решение закодировано в `ACCESS_GRANTED` / `ACCESS_DENIED`.

Изменение Neo4j и отправка Kafka не атомарны. Ошибка async producer callback
логируется; durable outbox отсутствует. Раздельные `auth-audit`/`auth-changes`
topics и строгая доставка остаются проектным изменением.

Для mutation events нужен `event_id`; consumer обязан выдерживать повторы и
out-of-order delivery. Если graph mutation и event должны фиксироваться вместе,
нужен durable outbox/inbox protocol либо эквивалентный механизм.

## Конфигурация

Поддерживаются переменные из `internal/config.py`, в частности:

```dotenv
NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password123
REDIS_HOST=redis
REDIS_PORT=6379
KAFKA_BOOTSTRAP=kafka:29092
GRPC_PORT=50051
```

`CACHE_TTL_SECONDS`, `KAFKA_AUDIT_TOPIC` и `CHANGES_TOPIC` сейчас не читаются
кодом. Development defaults не подходят для production secrets.

При запуске Python process с хоста container names `neo4j`, `redis` и `kafka`
нужно заменить адресами, доступными с host, либо опубликованными портами.

### Telemetry и defaults

| Переменная | Default кода |
|---|---|
| `NEO4J_URI`, `NEO4J_USER`, `NEO4J_PASSWORD` | `bolt://localhost:7687`, `neo4j`, `password123` (только dev) |
| `REDIS_HOST`, `REDIS_PORT` | `localhost`, `6379` |
| `KAFKA_BOOTSTRAP`, `GRPC_PORT` | `localhost:9092`, `50051` |
| `LOGGER_LEVEL`, `LOGGER_AS_JSON`, `LOGGER_ENABLE_OTLP` | `info`, `true`, `false` |
| `OTEL_METRICS_PUSH_INTERVAL`, `OTEL_EXPORTER_OTLP_ENDPOINT` | `60` секунд, `localhost:4317` |
| `OTEL_ENVIRONMENT`, `OTEL_SERVICE_NAME`, `OTEL_SERVICE_VERSION` | `development`, `authz`, `0.1.0` |

В Python переменная logger называется `LOGGER_ENABLE_OTLP`; в Go —
`LOGGER_ENABLE_OLTP`. Не унифицируйте написание в документации без изменения parser.
Decision cache использует Redis DB 0 и prefix `auth_decision` по constructor defaults;
TTL задан в service, отдельной env для него сейчас нет.

## Запуск

Полностью в Docker, из корня репозитория:

```bash
docker compose --profile services up --build -d authz
docker compose logs -f authz
```

### Локальный Python и зависимости в Docker

Из корня репозитория (Python 3.12; `pip` устанавливает зависимости из pyproject):

```bash
docker compose up -d neo4j redis kafka
docker compose --profile services stop authz
cd services/authz
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -e '.[test]'
NEO4J_URI=bolt://localhost:7687 REDIS_HOST=localhost KAFKA_BOOTSTRAP=localhost:9092 \
  OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 python -m entrypoints.server.main
```

Дождитесь готовности зависимостей в `docker compose ps`. Пароль Neo4j по умолчанию
`password123`; при изменении стенда задайте `NEO4J_PASSWORD` в окружении. Python
config не загружает `.env` автоматически. Entry point сам добавляет корень для
импортов `shared`; editable install обеспечивает импорты `internal`/`entrypoints`.

В другом терминале, из `services/authz`, с активированной той же `.venv`:

```bash
python -m entrypoints.cache_invalidator
```

Этот entrypoint создаёт зависимости с constructor defaults: Redis
`localhost:6379`, Kafka `localhost:9092`, topic `auth-changes`. Он **не читает**
`REDIS_HOST`/`KAFKA_BOOTSTRAP` из `Config`; произвольные container addresses требуют
изменения wiring. Поэтому команда подходит именно для зависимостей, доступных
по локальным портам, а не является готовым отдельным Compose deployment.

Docker image нужно собирать с root context:

```bash
docker build -f services/authz/Dockerfile .
```

### Проверка gRPC

Если установлен `grpcurl`, reflection и health можно проверить так:

```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

Для `Check`/`WriteTuple` поля и enum values нужно брать из
`shared/api/authz/v1/authz.proto`, чтобы примеры не расходились с generated API.

## Примеры использования

Команды изменяют только демонстрационное отношение в локальном графе. В публичном
API перед изменением прав потребуется аутентификация инициатора и ADMIN-check.

```bash
grpcurl -plaintext -d '{"subject":"user:550e8400-e29b-41d4-a716-446655440000","relation":"RELATION_HAS_PERMISSION","object":"bucket:docs-demo","level":"PERMISSION_LEVEL_ADMIN"}' \
  localhost:50051 opens3.authz.v1.PermissionService/WriteTuple
grpcurl -plaintext -d '{"subject":"user:550e8400-e29b-41d4-a716-446655440000","action":"ACTION_READ","object":"bucket:docs-demo"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Check
grpcurl -plaintext -d '{"subject":"user:550e8400-e29b-41d4-a716-446655440000"}' \
  localhost:50051 opens3.authz.v1.PermissionService/Read
grpcurl -plaintext -d '{"subject":"user:550e8400-e29b-41d4-a716-446655440000","relation":"RELATION_HAS_PERMISSION","object":"bucket:docs-demo"}' \
  localhost:50051 opens3.authz.v1.PermissionService/DeleteTuple
```

После удаления прежнее решение может оставаться в кэше. `allowed=false` — нормальный
ответ Check, а не gRPC `PERMISSION_DENIED`. Неизвестный/нулевой action или relation,
а также отсутствующий level для HAS_PERMISSION дают `INVALID_ARGUMENT`.
Остальные исключения не имеют общего явного mapping в servicer; не полагайтесь
на обещанный `UNAVAILABLE` для каждой ошибки инфраструктуры.

## Наблюдаемость и диагностика

Сервис создаёт traces, metrics и structured logs для transport и repository
operations. Их доставка зависит от OTEL endpoint и запущенного collector. Custom
HealthCheck проверяет Neo4j/Redis, standard health отражает lifecycle process.

## Разработка и тесты

Coverage (из каталога сервиса, после установки `.[test]`):

```bash
python -m pytest tests/unit -v --cov=internal --cov=entrypoints --cov-report=term-missing
```

Конфигурация integration fixtures находится в [tests/integration/conftest.py](tests/integration/conftest.py).
Недоступный Neo4j может приводить к skip: проверяйте summary тестов.

### Тесты

```bash
cd services/authz
python3 -m pytest tests/unit -v
python3 -m pytest tests/integration -v -m integration
```

Integration tests требуют Neo4j. Python-каталог `internal` является соглашением
структуры, а не языковым запретом импорта.

### Proto regeneration

Общие bindings генерируются из корня:

```bash
make generate
```

После изменения proto нужно обновить generated Go/Python outputs и проверить все
сервисы, которые импортируют `PermissionService`.

## Ограничения и дальнейшие работы

- текущая модель не наследует permission через `PARENT_OF`;
- cache invalidation не гарантирует мгновенный revoke;
- Neo4j mutation и Kafka event не атомарны;
- server и invalidation consumer требуют отдельных deployment units;
- внешний S3 resource/action mapping должен быть согласован в Gateway.

Варианты наследования ресурсов: materialized direct tuples, traversal
`PARENT_OF` во время Check или предварительно вычисленные permissions. Выбор
зависит от глубины graph, стоимости revoke и допустимой latency.

## Связанные документы

- [Обзор проекта](../../README.md) и [первый запуск](../../GETTING_STARTED.md).
- [Карта документации](../../docs/README.md).
- [Правила разработки](../../AGENTS.md).
