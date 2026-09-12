# Storage Service

Go gRPC-сервис immutable blob storage. Он работает с `blob_id` и не знает о S3
buckets, object keys, users или правах доступа.

## Назначение и возможности

Обычная и multipart-загрузка, чтение целиком и по диапазону, удаление blob и отмена загрузки. Это внутренний data path; каталог объектов ведёт Metadata.

## Архитектура и зависимости

```text
gRPC handler → Storage service → filesystem repository
                     ├── ID generation
                     ├── stream validation
                     └── multipart orchestration
```

Handler переводит protobuf streams в service calls. Service управляет IDs и
последовательностью операций. Repository отвечает за temporary files, final
paths, чтение ranges и cleanup.

## Структура проекта

```text
cmd/server/                         entrypoint
internal/app/                       wiring и lifecycle
internal/handler/storage/           gRPC transport
internal/service/storage/           single-part и multipart logic
internal/repository/storage/        filesystem implementation
internal/config/                    environment configuration
pkg/mocks/                          generated mocks
```

## API

Source of truth: `shared/api/storage/v1/storage.proto`.

| RPC | Тип | Назначение |
|---|---|---|
| `StoreObject` | client stream | сохранить blob, вернуть UUID и full-content MD5 |
| `RetrieveObject` | server stream | читать blob целиком или по offset/length |
| `DeleteObject` | unary | идемпотентно удалить blob |
| `InitiateMultipartUpload` | unary | создать multipart session/upload ID |
| `UploadPart` | client stream | сохранить/перезаписать part |
| `CompleteMultipartUpload` | unary | проверить список parts и собрать blob |
| `AbortMultipartUpload` | unary | идемпотентно очистить session |
| `HealthCheck` | unary | проверить filesystem repository |

Client-streaming StoreObject/UploadPart используют первое сообщение `header`,
затем сообщения `chunk`. Пустой объект поддерживается.

Handler отклоняет нарушенный порядок messages. Header может содержать `data`;
поле bytes представлено base64 при использовании grpcurl JSON. `RetrieveObject` возвращает server
stream; offset и length задают range, а точные edge cases определяются protobuf и
service validation.

## Данные и основные сценарии

### Файловый commit

Обычная запись получает blob ID в service до начала repository write. Repository
пишет staging file, вызывает `Sync`, закрывает его и выполняет rename в final path.
Final file появляется после rename, а не при первом chunk. Проверка ожидаемого
size выполняется service **после** repository commit; при mismatch он пытается
удалить blob. Ошибка cleanup может оставить orphan, несмотря на ошибочный RPC.

Atomic rename гарантируется только внутри одной filesystem. Текущий код не делает
fsync каталога после rename, поэтому документация не обещает полную durability при
аварийной потере питания.

#### Layout на диске

```text
DATA_DIR/<shard>/<blob_id>                         final blob
MULTIPART_DIR/uploads/<upload_id>/                 session и parts
MULTIPART_DIR/completed/<shard>/<upload_id>.json   completion marker
```

Sharding ограничивает число entries в одном каталоге. Metadata хранит blob IDs,
но не строит filesystem paths самостоятельно.

Multipart upload создаёт session заранее. В текущем repository итоговый blob ID
совпадает с upload ID. Complete сохраняет completion metadata для retry и затем
best-effort очищает session внутри Storage; он не ждёт callback от Metadata.

### S3 и multipart

Storage возвращает MD5 полного собранного blob. Это внутренний checksum, а не
универсальный внешний multipart ETag. Gateway должен отдельно реализовать выбранную
S3 checksum/ETag semantics.

S3 minimum part size проверяется на Complete для всех выбранных частей, кроме
последней: при отдельном UploadPart ещё неизвестно, какая часть окажется последней.
Storage валидирует свой внутренний контракт; полная S3-валидация относится к
Gateway/Metadata orchestration.

Multipart part можно загрузить повторно по тому же номеру; новая успешная запись
заменяет прежнюю. `Complete` получает выбранный упорядоченный список, проверяет
checksums и создаёт final blob. Completion marker хранит terminal result для retry
после потерянного ответа.

## Конфигурация

### Полный перечень переменных Go-конфигурации

Default ниже относится к коду, а не к development `.env`.

| Переменная | Default / требование | Раздел конфигурации |
|---|---|---|
| `GRPC_HOST` | задать явно | [grpc.go](internal/config/env/grpc.go) |
| `GRPC_PORT` | задать явно | [grpc.go](internal/config/env/grpc.go) |
| `LOGGER_LEVEL` | задать явно | [logger.go](internal/config/env/logger.go) |
| `LOGGER_AS_JSON` | задать явно | [logger.go](internal/config/env/logger.go) |
| `LOGGER_ENABLE_OLTP` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_SERVICE_NAME` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_ENVIRONMENT` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_SERVICE_VERSION` | задать явно | [metrics.go](internal/config/env/metrics.go) |
| `OTEL_METRICS_PUSH_TIMEOUT` | задать явно | [metrics.go](internal/config/env/metrics.go) |
| `RATE_LIMITER_LIMIT` | `100` | [rate_limiter.go](internal/config/env/rate_limiter.go) |
| `RATE_LIMITER_PERIOD` | `1s` | [rate_limiter.go](internal/config/env/rate_limiter.go) |
| `DATA_DIR` | `/data/blobs` | [storage.go](internal/config/env/storage.go) |
| `MULTIPART_DIR` | `/data/staging` | [storage.go](internal/config/env/storage.go) |

Основные значения находятся в `services/storage/.env`:

```dotenv
GRPC_PORT=50053
DATA_DIR=/data/blobs
MULTIPART_DIR=/data/staging
```

Для гарантированного rename staging/final paths должны находиться на одной
filesystem.

Полный набор параметров и validation rules находится в `internal/config`.

## Запуск

Из корня репозитория:

```bash
docker compose --profile services up --build -d storage
docker compose logs -f storage
```

Локально:

```bash
cd services/storage
DATA_DIR="/tmp/opens3-storage-dev/blobs" MULTIPART_DIR="/tmp/opens3-storage-dev/staging" \
  OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 go run ./cmd/server
```

## Примеры использования

Для небольшого демонстрационного blob `abc` (`YWJj` — base64 поля bytes):

```bash
grpcurl -plaintext -d '{"header":{"size":"3","content_type":"text/plain","data":"YWJj"}}' \
  localhost:50053 opens3.storage.v1.DataStorageService/StoreObject
```

Подставьте `blobId` из ответа:

```bash
grpcurl -plaintext -d '{"blob_id":"<blob-id>","offset":"0","length":"0"}' \
  localhost:50053 opens3.storage.v1.DataStorageService/RetrieveObject
grpcurl -plaintext -d '{"blob_id":"<blob-id>"}' \
  localhost:50053 opens3.storage.v1.DataStorageService/DeleteObject
```

Retrieve возвращает bytes в base64; `length=0` означает читать до конца.
Header может содержать первые байты; последующие сообщения — только chunk.
Для больших файлов используйте generated streaming client с bounded buffering.
Готовый multipart smoke-клиент проверяет initiate → upload parts → complete →
retrieve, сверяет байты/MD5 и оставляет итоговый blob для осмотра:

```bash
cd services/storage
go run ./cmd/multipart-smoke -addr localhost:50053
```

Это проверка внутреннего multipart API, а не S3 minimum part size.

## Наблюдаемость и диагностика

Сервис создаёт structured logs, traces и metrics через общий Go kit. Экспорт
зависит от OTEL collector. Ошибки repository преобразуются в domain/gRPC errors;
клиент должен различать invalid input, missing blob/session и internal I/O error.

После cancellation или ошибки записи temporary file должен быть удалён. Ошибка
cleanup после успешного rename требует logs/metrics и reconciliation, но не должна
делать committed bytes неизвестными coordinator.

Тестовый набор включает unit и filesystem/component scenarios: empty/large
streams, size mismatch, range reads, delete, multipart overwrite/complete/abort и
retry. Crash durability и multi-node replication требуют отдельных сред.

## Разработка и тесты

### Тесты

```bash
cd services/storage
go test ./...
```

Инвентаризация тестов: [tests.md](tests.md). Standard gRPC Health выставляется в
SERVING при старте; custom `DataStorageService.HealthCheck` отдельно проверяет
filesystem. Эти два endpoint нельзя считать одной и той же readiness-проверкой.

### Proto generation

Из корня:

```bash
make generate-storage-go
make generate-storage-py
```

Либо сгенерировать все bindings:

```bash
make generate
```

`go.work` подменяет опубликованные module versions локальными каталогами во время
работы в workspace. При этом `services/storage/go.mod` всё равно содержит обычную
module dependency на shared package.

## Ограничения и дальнейшие работы

### Что пока не реализовано

- репликация blob на несколько Storage nodes;
- placement service и consistent hashing;
- Kafka consumer для object cleanup;
- внешний S3 HTTP API;
- durable связь Storage completion с Metadata finalize.

Нельзя вызвать StoreObject на нескольких узлах и ожидать один blob ID: текущий
request не принимает заранее выбранный ID, а каждый service сам создаёт UUID.

Варианты будущего cleanup: синхронный вызов для простого стенда, Kafka consumer с
at-least-once delivery либо reconciliation worker по durable catalog. Для
асинхронных вариантов нужны точные blob/version IDs и идемпотентность.

### Варианты распределённого data path

- Gateway/coordinator пишет replicas параллельно;
- Storage nodes передают поток по chain;
- fan-out выполняет отдельный data proxy.

Текущий single-node API остаётся базовым blob interface. Изменения IDs, fencing,
quorum и repair описаны в
[`docs/distributed-storage-architecture.md`](../../docs/distributed-storage-architecture.md).

## Связанные документы

- [Обзор проекта](../../README.md) и [первый запуск](../../GETTING_STARTED.md).
- [Карта документации](../../docs/README.md).
- [Правила разработки](../../AGENTS.md).
