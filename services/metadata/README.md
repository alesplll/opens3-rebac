# Metadata Service

Go 1.24.1 gRPC-сервис каталога объектного хранилища: buckets, object keys, versions
и ссылки на blobs в PostgreSQL. Порт локального стенда — `50052`.

## Назначение и возможности

- создание, получение, перечисление и удаление buckets;
- регистрация committed blob-версии и переключение current version;
- получение текущей или конкретной committed версии;
- постраничный список объектов с фильтром prefix;
- удаление метаданных и публикация событий удаления;
- custom проверка PostgreSQL/Kafka, standard gRPC health, metrics и tracing.

Сервис не хранит байты, не проверяет JWT/ReBAC и не вызывает Storage. Переданный
`blob_id` связывается с версией без проверки наличия файла на Storage.

## Архитектура и зависимости

```text
gRPC MetadataService → handler/metadata
                         ├── service/bucket → repository/bucket → PostgreSQL
                         └── service/object → repository/object → PostgreSQL
                                  └── Kafka producer (delete events)
```

Service задаёт транзакционные границы, repository выполняет SQL, handler переводит
запросы/ответы и ошибки. Общие DB client, transaction manager, Kafka producer и
observability находятся в `shared/pkg/go-kit`.

| Зависимость | В Compose | С хоста | Назначение |
|---|---|---|---|
| PostgreSQL Metadata | `postgres-metadata:5432` | `localhost:5433` | каталог |
| Kafka | `kafka:29092` | `localhost:9092` | события удаления |
| OTEL Collector | `otel-collector:4317` | `localhost:4317` | экспорт telemetry |

Users использует другую PostgreSQL. Между БД Users и Metadata нет общего FK или
транзакции: `owner_id` хранится как UUID.

## Структура проекта

```text
cmd/server/                        точка входа
internal/app/                      wiring и lifecycle
internal/config/                   env configuration
internal/handler/metadata/         transport всех RPC
internal/service/bucket/           bucket operations
internal/service/object/           version/catalog operations
internal/repository/bucket/        SQL buckets
internal/repository/object/        SQL objects/versions
internal/model/                    domain model
pkg/mocks/                         minimock dependencies
migrations/                        PostgreSQL schema (goose)
```

## API

Wire-контракт: [metadata.proto](../../shared/api/metadata/v1/metadata.proto),
`opens3.metadata.v1.MetadataService`. Комментарии proto о будущем Gateway/cleanup
не подтверждают наличие этих consumers; текущее поведение описано ниже.

| RPC | Вход / результат |
|---|---|
| `CreateBucket` | name + owner_id → bucket_id + created_at |
| `GetBucket` | bucket_name → BucketInfo |
| `HeadBucket` | bucket_name → exists, bucket_id, owner_id |
| `ListBuckets` | owner_id → список принадлежащих ему buckets |
| `DeleteBucket` | bucket_name → success после проверки числа objects |
| `CreateObjectVersion` | bucket_name, key, blob_id, size_bytes, etag, content_type → object_id, version_id |
| `GetObjectMeta` | bucket_name, key, необязательный version_id → blob_id и metadata |
| `ListObjects` | bucket_name, prefix, continuation_token, max_keys → объекты и следующий token |
| `DeleteObjectMeta` | bucket_name + key → object_id, текущий blob_id, success |
| `HealthCheck` | пустой запрос → SERVING / NOT_SERVING по PostgreSQL и Kafka |

`last_modified` сейчас берётся из `versions.created_at`. ListObjects выбирает
active objects с current committed blob version, сортирует по key; token — последний
key предыдущей страницы. `max_keys <= 0` или `> 1000` нормализуется к 1000.
Если строк ровно `max_keys`, `is_truncated=true` даже без следующей строки —
последующий запрос может вернуть пустую страницу. Prefix использует SQL LIKE,
поэтому `%`/`_` пока имеют смысл SQL wildcard, а не буквальных символов S3 prefix.

## Данные и основные сценарии

### Схема каталога

```text
buckets (id, name UNIQUE, owner_id)
  └── objects (id, bucket_id, key, status,
               current_version_id, pending_version_id; UNIQUE(bucket_id, key))
        └── versions (id, object_id, version_number, kind, state,
                      blob_id, size_bytes, etag, content_type,
                      created_at, committed_at, aborted_at)
```

[Миграция](migrations/20260423000000_init.sql) определяет `kind=blob/delete_marker`,
`state=pending/committed/aborted`, deferred FK указателей версий и уникальность
`(object_id, version_number)`. Наличие полей pending/delete_marker не означает
реализованные RPC finalize/abort/versioning lifecycle.

### Регистрация версии

Текущий `CreateObjectVersion` выполняет в одной READ COMMITTED транзакции:

1. Upsert объекта по bucket/key.
2. Блокировку строки объекта и выделение `MAX(version_number)+1`.
3. Insert committed blob version с `committed_at`.
4. Обновление `current_version_id`.

Блокировка и INSERT должны оставаться в одной транзакции. Старые версии не
удаляются при overwrite. Повтор CreateObjectVersion создаёт новую версию:
идемпотентного operation ID пока нет, ETag его не заменяет.

Будущий coordinator сначала получает успешный Storage commit, затем регистрирует
Metadata version и только после этого возвращает внешний успех. Это целевой
межсервисный flow; атомарной транзакции между файлом и PostgreSQL сейчас нет.

### Удаление и Kafka

| Topic из development config | Key | JSON payload | Текущая доставка |
|---|---|---|---|
| `object-deleted` | object_id | `object_id`, `blob_id` | после DELETE; best-effort |
| `bucket-deleted` | bucket_id | `bucket_id`, `bucket_name` | после DB commit; best-effort |

При ошибке publish RPC возвращает ошибку, хотя данные уже удалены. Повтор не
гарантирует восстановления события. Storage/AuthZ cleanup consumers для этих
событий пока не реализованы; outbox остаётся отдельной работой.

DeleteObjectMeta удаляет object и каскадно все его versions, но возвращает лишь
текущий blob_id. Это не delete marker и не полная очистка физических версий:
ссылки на старые blobs теряются. До version-aware GC не полагайтесь на удаление
каталога как на освобождение всех байтов.

## Конфигурация

Go загружает `.env` из текущего каталога; можно передать `-config-path`. Уже
заданные переменные окружения имеют приоритет. Compose читает сервисный
[.env](.env); значения в нём относятся к локальному стенду.

| Переменные | Значение / назначение |
|---|---|
| `GRPC_HOST`, `GRPC_PORT` | обязательные; в стенде `0.0.0.0`, `50052` |
| `PG_DSN` | необязательная готовая строка подключения, приоритет над полями PG |
| `PG_HOST`, `PG_PORT_INNER`, `PG_DATABASE_NAME`, `PG_USER`, `PG_PASSWORD` | обязательный полный набор, если PG_DSN не задан |
| `PG_SSL_MODE`, `PG_TIMEOUT`, `PG_LOGGING` | defaults кода `disable`, `5s`, `false` |
| `KAFKA_BOOTSTRAP` | обязательный, Compose `kafka:29092` |
| `KAFKA_OBJECT_DELETED_TOPIC`, `KAFKA_BUCKET_DELETED_TOPIC` | обязательные; `object-deleted`, `bucket-deleted` |
| `RATE_LIMITER_LIMIT`, `RATE_LIMITER_PERIOD` | defaults `100`, `1s`, локальный limiter |
| `LOGGER_LEVEL`, `LOGGER_AS_JSON`, `LOGGER_ENABLE_OLTP` | обязательные настройки Go logger; написание **OLTP** соответствует коду |
| `OTEL_SERVICE_NAME`, `OTEL_SERVICE_VERSION`, `OTEL_ENVIRONMENT` | обязательные атрибуты telemetry |
| `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_METRICS_PUSH_TIMEOUT` | обязательные endpoint и timeout |

Парсинг и полный набор: [internal/config/env](internal/config/env). Credentials
базы должны совпадать с PostgreSQL и migrator; смена env не меняет пароль уже
инициализированного volume.

## Запуск

### Docker

Из корня репозитория:

```bash
docker compose --profile services up --build -d metadata
docker compose ps -a migrator-metadata metadata
docker compose logs -f metadata
```

Compose дожидается PostgreSQL/Kafka и успешного `migrator-metadata`.
Приложение само миграции не запускает.

### Локальный Go и зависимости в Docker

Из корня:

```bash
docker compose --profile services up --build -d migrator-metadata kafka
docker compose --profile services stop metadata
cd services/metadata
PG_HOST=localhost PG_PORT_INNER=5433 KAFKA_BOOTSTRAP=localhost:9092 \
  OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 go run ./cmd/server
```

Перед запуском дождитесь завершения мигратора с кодом 0 и готовности Kafka.
Для экспорта telemetry поднимите `make up-observability` из корня.

## Примеры использования

`grpcurl` работает с reflection. Создание демонстрационного bucket:

```bash
grpcurl -plaintext -d '{"name":"docs-demo","owner_id":"550e8400-e29b-41d4-a716-446655440000"}' \
  localhost:50052 opens3.metadata.v1.MetadataService/CreateBucket
grpcurl -plaintext -d '{"bucket_name":"docs-demo"}' \
  localhost:50052 opens3.metadata.v1.MetadataService/GetBucket
```

Сначала сохраните blob по примеру [Storage](../storage/README.md), затем подставьте
полученные blobId/checksumMd5; размер должен соответствовать файлу:

```bash
grpcurl -plaintext -d '{"bucket_name":"docs-demo","key":"example.txt","blob_id":"<blob-id>","size_bytes":"3","etag":"<checksum-md5>","content_type":"text/plain"}' \
  localhost:50052 opens3.metadata.v1.MetadataService/CreateObjectVersion
grpcurl -plaintext -d '{"bucket_name":"docs-demo","key":"example.txt"}' \
  localhost:50052 opens3.metadata.v1.MetadataService/GetObjectMeta
grpcurl -plaintext -d '{"bucket_name":"docs-demo","max_keys":10}' \
  localhost:50052 opens3.metadata.v1.MetadataService/ListObjects
```

Удаление demo metadata через DeleteObjectMeta не удалит blob автоматически:
cleanup-сценарий ещё не собран. Для ручной уборки сохраните blob_id и удалите blob
отдельным Storage DeleteObject, затем metadata и пустой bucket.

## Наблюдаемость и диагностика

```bash
grpcurl -plaintext localhost:50052 grpc.health.v1.Health/Check
grpcurl -plaintext -d '{}' localhost:50052 opens3.metadata.v1.MetadataService/HealthCheck
```

Первый endpoint отражает lifecycle статус процесса; второй выполняет PostgreSQL
Ping и Kafka RefreshMetadata. TCP-доступность не доказывает готовность зависимостей.

При ошибке подключения проверьте container/host адреса; при missing tables —
логи `migrator-metadata`; при ошибке delete/publish проверьте состояние каталога
перед повтором. Structured logs, metrics и spans экспортируются через Go kit.

## Разработка и тесты

Unit tests (из сервиса):

```bash
cd services/metadata
go test ./...
```

Repository integration tests (из корня; отдельная тестовая БД):

```bash
make test-metadata-integration-local
make down-e2e
```

Без автоматического запуска контейнера: `make up-e2e`, затем
`make test-metadata-integration`. Suite пересоздаёт схему и чистит таблицы —
используйте конфигурацию [e2e](../../e2e/README.md), а не development/рабочую БД.
Тест выдачи version_number: [insert_version_integration_test.go](internal/repository/object/tests/insert_version_integration_test.go).

После изменения wire schema из корня выполните `make install-deps`, затем
`make generate-metadata-go` / `make generate-metadata-py` с установленным `protoc`.
Generated outputs вручную не редактируются.

## Ограничения и дальнейшие работы

- pending/finalize, delete markers и полный version lifecycle ещё не реализованы;
- outbox, version-aware GC и безопасные retries требуют отдельного протокола;
- нет межсервисной аутентификации или проверки owner_id по identity;
- prefix/pagination имеют описанные выше отличия от внешнего S3-контракта;
- проверка непустоты bucket и конкурентное создание объектов требуют согласования
  lifecycle; наличие транзакции само по себе не доказывает нужную сериализацию.

## Связанные документы

- [Обзор проекта](../../README.md) и [первый запуск](../../GETTING_STARTED.md).
- [Карта документации](../../docs/README.md).
- [Правила разработки](../../AGENTS.md).
- [Фактологический аудит: C01–C03](../../docs/factual-audit-2026-09-10.md).
- [План согласования Storage и Metadata](../../docs/storage-service-implementation-plan.md).
