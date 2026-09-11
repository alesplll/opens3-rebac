# AuthZ Service

Python 3.12 gRPC-сервис Relationship-Based Access Control. Хранит graph в Neo4j,
кэширует решения в Redis и публикует audit/tuple events в Kafka.

## Архитектура

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

## Реализованный API

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

Публикация audit event не входит в атомарную транзакцию graph read. Ошибка Kafka
не должна менять сам результат авторизации, но влияет на полноту audit trail.

## Модель отношений

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

## Кэш и отзыв доступа

Решения сохраняются в Redis на 30 секунд. Основной Docker entrypoint запускает
только gRPC server; `entrypoints/cache_invalidator.py` нужно разворачивать
отдельно, если используется Kafka invalidation.

Даже с consumer текущая инвалидация best-effort. Удаление group membership не
перечисляет все resource decisions, полученные через группу, а конкурентный
graph-check способен записать старый ALLOW после очистки. Поэтому нельзя обещать
немедленный отзыв: верхняя практическая граница задаётся TTL, пока не реализованы
dependency-aware invalidation либо versioned epochs.

Не увеличивайте/sliding-extend ALLOW TTL без независимого hard deadline: потеря
invalidation тогда может продлевать уже отозванный доступ неограниченно.

Безопасные варианты развития: dependency-aware invalidation всех зависимых
решений, epochs/generations в cache key либо короткий bounded TTL как временная
граница stale access. Выбор и race tests зафиксированы в issue #67.

## Kafka

Текущая конфигурация использует один producer с topic `auth-changes` по умолчанию.
Туда отправляются tuple events и authorization decisions. Payload решения содержит
`event_type`, `subject`, `action`, `object`, `allowed` и `timestamp`; он не содержит
документированные ранее `from_cache`, `level` или `timestamp_ms`.

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

## Запуск

Полностью в Docker, из корня репозитория:

```bash
docker compose --profile services up --build -d authz
docker compose logs -f authz
```

Локальный процесс при уже запущенном Docker AuthZ конфликтует за порт. Для local
run сначала остановите именно контейнер `authz`, оставив зависимости:

```bash
docker compose --profile services stop authz
cd services/authz
python3 -m entrypoints.server.main
```

Отдельный invalidation process запускается entrypoint
`entrypoints/cache_invalidator.py` с теми же доступными Redis/Kafka addresses.

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

## Observability

Сервис создаёт traces, metrics и structured logs для transport и repository
operations. Их доставка зависит от OTEL endpoint и запущенного collector. Custom
HealthCheck проверяет Neo4j/Redis, standard health отражает lifecycle process.

## Тесты

```bash
cd services/authz
python3 -m pytest tests/unit -v
python3 -m pytest tests/integration -v -m integration
```

Integration tests требуют Neo4j. Python-каталог `internal` является соглашением
структуры, а не языковым запретом импорта.

## Proto regeneration

Общие bindings генерируются из корня:

```bash
make generate
```

После изменения proto нужно обновить generated Go/Python outputs и проверить все
сервисы, которые импортируют `PermissionService`.

## Ограничения и дальнейшие варианты

- текущая модель не наследует permission через `PARENT_OF`;
- cache invalidation не гарантирует мгновенный revoke;
- Neo4j mutation и Kafka event не атомарны;
- server и invalidation consumer требуют отдельных deployment units;
- внешний S3 resource/action mapping должен быть согласован в Gateway.

Варианты наследования ресурсов: materialized direct tuples, traversal
`PARENT_OF` во время Check или предварительно вычисленные permissions. Выбор
зависит от глубины graph, стоимости revoke и допустимой latency.
