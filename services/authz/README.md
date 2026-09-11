# AuthZ Service

Python 3.12 gRPC-сервис Relationship-Based Access Control. Хранит graph в Neo4j,
кэширует решения в Redis и публикует audit/tuple events в Kafka.

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

## Кэш и отзыв доступа

Решения сохраняются в Redis на 30 секунд. Основной Docker entrypoint запускает
только gRPC server; `entrypoints/consumer` нужно разворачивать отдельно, если
используется Kafka invalidation.

Даже с consumer текущая инвалидация best-effort. Удаление group membership не
перечисляет все resource decisions, полученные через группу, а конкурентный
graph-check способен записать старый ALLOW после очистки. Поэтому нельзя обещать
немедленный отзыв: верхняя практическая граница задаётся TTL, пока не реализованы
dependency-aware invalidation либо versioned epochs.

Не увеличивайте/sliding-extend ALLOW TTL без независимого hard deadline: потеря
invalidation тогда может продлевать уже отозванный доступ неограниченно.

## Kafka

Текущая конфигурация использует один producer с topic `auth-changes` по умолчанию.
Туда отправляются tuple events и authorization decisions. Payload решения содержит
`event_type`, `subject`, `action`, `object`, `allowed` и `timestamp`; он не содержит
документированные ранее `from_cache`, `level` или `timestamp_ms`.

Изменение Neo4j и отправка Kafka не атомарны. Ошибка async producer callback
логируется; durable outbox отсутствует. Раздельные `auth-audit`/`auth-changes`
topics и строгая доставка остаются проектным изменением.

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

Docker image нужно собирать с root context:

```bash
docker build -f services/authz/Dockerfile .
```

## Тесты

```bash
cd services/authz
python3 -m pytest tests/unit -v
python3 -m pytest tests/integration -v -m integration
```

Integration tests требуют Neo4j. Python-каталог `internal` является соглашением
структуры, а не языковым запретом импорта.
