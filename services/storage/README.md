# Storage Service (Data Node)

Микросервис хранения бинарных объектов (blob) на файловой системе.
Является частью распределённого S3-совместимого объектного хранилища **OpenS3**.

Сервис **не знает** о бакетах, ключах объектов, правах доступа или S3 API —
он работает только с `blob_id` (UUID) и байтами. Вызывается исключительно из Gateway.

---

## Оглавление

- [Архитектура](#архитектура)
- [Структура проекта](#структура-проекта)
- [gRPC API](#grpc-api)
- [Конфигурация](#конфигурация)
- [Запуск](#запуск)
- [Хранилище на диске](#хранилище-на-диске)
- [Dependency Injection](#dependency-injection)
- [Обработка ошибок](#обработка-ошибок)
- [Observability](#observability)
- [Kafka-интеграция](#kafka-интеграция)
- [Разработка](#разработка)
- [TODO](#todo)

---

## Архитектура

Трёхслойная архитектура по аналогии с остальными Go-сервисами проекта:

```
gRPC request
    │
    ▼
┌──────────────────────────────────────────┐
│  Middleware (rate limiter → metrics →     │
│  validation → tracing)                   │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│  Handler     (internal/handler/storage/) │  ← transport layer:
│              разбирает первый message     │     стрима, валидирует streaming
│              и прокидывает byte-stream    │     contract, делегирует в Service
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│  Service     (internal/service/storage/) │  ← application/domain layer:
│              валидирует команды,          │     назначает id, логирует,
│              оркестрирует Repository      │     обновляет метрики
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│  Repository  (internal/repository/       │  ← persistence layer:
│              storage/)                   │     staging, атомарная запись,
│                                          │     чтение/удаление blob-файлов
└──────────────────────────────────────────┘
```

Каждый слой определён через **интерфейс** и подключается через DI-контейнер.

---

## Структура проекта

```
services/storage/
├── cmd/server/
│   └── main.go                              # Точка входа
├── internal/
│   ├── app/
│   │   ├── app.go                           # Жизненный цикл приложения
│   │   └── service_provider.go              # DI-контейнер (lazy init)
│   ├── config/
│   │   ├── config.go                        # Загрузка конфигурации из .env
│   │   ├── interfaces.go                    # Интерфейсы конфигов
│   │   └── env/
│   │       ├── grpc.go                      # GRPC_HOST, GRPC_PORT
│   │       ├── storage.go                   # DATA_DIR, MULTIPART_DIR
│   │       ├── logger.go                    # LOGGER_*
│   │       ├── tracing.go                   # OTEL tracing
│   │       ├── metrics.go                   # OTEL metrics
│   │       └── rate_limiter.go              # RATE_LIMITER_*
│   ├── handler/storage/                     # Слой 1: gRPC-хэндлеры
│   │   ├── handler.go                       # Конструктор, embed Unimplemented
│   │   ├── upload_streams.go                # Контракт и разбор client-streaming запросов
│   │   ├── store_object.go                  # Client-streaming: приём blob
│   │   ├── retrieve_object.go               # Server-streaming: отдача blob
│   │   ├── delete_object.go                 # Unary: удаление blob
│   │   ├── initiate_multipart.go            # Unary: старт multipart
│   │   ├── upload_part.go                   # Client-streaming: приём части
│   │   ├── complete_multipart.go            # Unary: склейка частей
│   │   ├── abort_multipart.go               # Unary: отмена multipart
│   │   └── health_check.go                  # Unary: проверка здоровья
│   ├── service/
│   │   └── service.go                       # Интерфейс StorageService
│   ├── service/storage/                     # Слой 2: бизнес-логика
│   │   ├── service.go                       # Конструктор
│   │   ├── store_object.go
│   │   ├── retrieve_object.go
│   │   ├── delete_object.go
│   │   ├── initiate_multipart.go
│   │   ├── upload_part.go
│   │   ├── complete_multipart.go
│   │   ├── abort_multipart.go
│   │   └── health_check.go
│   ├── repository/
│   │   └── repository.go                    # Интерфейс StorageRepository
│   ├── repository/storage/                  # Слой 3: доступ к FS
│   │   ├── repository.go                    # Конструктор (принимает StorageConfig)
│   │   ├── store_blob.go                    # Запись файла на диск
│   │   ├── multipart.go                     # Multipart session, part upload, final assemble
│   │   ├── retrieve_blob.go                 # Чтение файла с диска
│   │   ├── delete_blob.go                   # Удаление файла
│   │   ├── write_helpers.go                 # Атомарная запись и temp-файлы
│   │   └── health.go                        # Проверка доступности DATA_DIR
│   ├── model/
│   │   └── blob.go                          # Доменные модели: BlobMeta, PartInfo
│   └── errors/domain_errors/
│       └── apperrors.go                     # Доменные ошибки
├── .env                                     # Переменные окружения (development)
├── go.mod
└── go.sum
```

---

## gRPC API

**Пакет:** `opens3.storage.v1`
**Сервис:** `DataStorageService`
**Порт:** `50053` (по умолчанию)
**Proto-файл:** `shared/api/storage/v1/storage.proto`

### Методы

| Метод | Тип | Описание |
|---|---|---|
| `StoreObject` | Client-streaming | Принимает поток чанков → сохраняет blob → возвращает `blob_id` + MD5 |
| `RetrieveObject` | Server-streaming | Читает blob → стримит чанки (поддержка Range) |
| `DeleteObject` | Unary | Удаляет blob по `blob_id` (идемпотентно) |
| `InitiateMultipartUpload` | Unary | Создаёт сессию multipart-загрузки → возвращает `upload_id` |
| `UploadPart` | Client-streaming | Принимает чанки одной части → сохраняет → возвращает MD5 части |
| `CompleteMultipartUpload` | Unary | Склеивает все части → возвращает итоговый `blob_id` + MD5 |
| `AbortMultipartUpload` | Unary | Отменяет multipart, удаляет временные файлы (идемпотентно) |
| `HealthCheck` | Unary | Проверяет доступность `DATA_DIR` и свободное место |

### Потоковые RPC — как это работает

**Client-streaming** (`StoreObject`, `UploadPart`):
```
Gateway                          Storage
  │                                │
  │──── StoreObjectRequest ──────► │  (data + size + content_type в первом чанке)
  │──── StoreObjectRequest ──────► │  (data)
  │──── StoreObjectRequest ──────► │  (data)
  │──── EOF ─────────────────────► │
  │                                │  потоково читает чанки и пишет на диск
  │◄─── StoreObjectResponse ────── │  (blob_id, checksum_md5)
```

Для `StoreObject` streaming contract:
- первое сообщение может содержать первые байты файла и metadata: `size`, `content_type`
- все последующие сообщения должны содержать только `data`
- пустой объект всё равно требует хотя бы одно сообщение в stream-е
- если `size` или `content_type` пришли не в первом сообщении, сервер вернёт `INVALID_ARGUMENT`

Для `UploadPart` действуют те же streaming semantics:
- первое сообщение обязано содержать `upload_id` и `part_number`, и может одновременно содержать `data`
- все последующие сообщения должны содержать только `data`
- если `upload_id` или `part_number` повторно присланы после первого сообщения, сервер вернёт `INVALID_ARGUMENT`
- данные части пишутся потоково, без буферизации всего part в памяти handler-а
- части можно загружать в любом порядке; порядок фактической загрузки не влияет на итоговую сборку
- при `CompleteMultipartUpload` список `parts` должен быть отсортирован по возрастанию `part_number`
- текущая реализация не требует непрерывной последовательности `1..N`: сервис собирает итоговый blob из тех `part_number`, которые явно переданы в `CompleteMultipartUpload` и реально существуют на диске

**Server-streaming** (`RetrieveObject`):
```
Gateway                          Storage
  │                                │
  │──── RetrieveObjectRequest ───► │  (blob_id, offset, length)
  │                                │
  │◄─── RetrieveObjectResponse ─── │  (data + total_size в первом чанке)
  │◄─── RetrieveObjectResponse ─── │  (data)
  │◄─── RetrieveObjectResponse ─── │  (data)
  │◄─── EOF ────────────────────── │
```

Размер чанка при отдаче: **8 MB** (константа `chunkSize` в `handler/storage/retrieve_object.go`).

### Когда именно появляются blob и part

- `StoreObject`: staging-директория и `manifest.json` создаются сразу при вызове метода repository; финальный blob в `DATA_DIR/.../<blob_id>` появляется только после полного чтения stream-а и атомарного `rename`.
- `InitiateMultipartUpload`: сразу создаются директория сессии `MULTIPART_DIR/uploads/<upload_id>/` и manifest с metadata upload-а.
- `UploadPart`: файл `part_<number>` создаётся во время записи входного stream-а; фактически part появляется, когда repository начинает читать и писать байты, а не в момент открытия RPC.
- `CompleteMultipartUpload`: итоговый blob сначала собирается во временный файл и становится видимым только после финального `rename`; затем multipart staging удаляется best-effort.

---

## Конфигурация

Конфигурация загружается из переменных окружения. Файл `.env` используется для локальной разработки.

### Обязательные переменные

| Переменная | Описание | Пример |
|---|---|---|
| `GRPC_HOST` | Хост gRPC сервера | `0.0.0.0` |
| `GRPC_PORT` | Порт gRPC сервера | `50053` |
| `LOGGER_LEVEL` | Уровень логирования | `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `LOGGER_AS_JSON` | JSON-формат логов | `true` / `false` |
| `LOGGER_ENABLE_OLTP` | Экспорт логов в OTEL | `true` / `false` |
| `OTEL_SERVICE_NAME` | Имя сервиса в OTEL | `storage_server` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP Collector endpoint | `otel-collector:4317` |
| `OTEL_ENVIRONMENT` | Окружение | `development` |
| `OTEL_SERVICE_VERSION` | Версия сервиса | `1.0.0` |
| `OTEL_METRICS_PUSH_TIMEOUT` | Таймаут отправки метрик | `1s` |

### Опциональные переменные

| Переменная | По умолчанию | Описание |
|---|---|---|
| `DATA_DIR` | `/data/blobs` | Корневая директория для финальных blob (`<shard>/<blob_id>`) |
| `MULTIPART_DIR` | `/data/staging` | Корневая директория staging-зоны (`uploads/` + `completed/`) |
| `RATE_LIMITER_LIMIT` | `100` | Максимум запросов в окно |
| `RATE_LIMITER_PERIOD` | `1s` | Окно rate limiter |

### Пример `.env`

```env
GRPC_HOST=0.0.0.0
GRPC_PORT=50053

DATA_DIR=/data/blobs
MULTIPART_DIR=/data/staging

LOGGER_LEVEL=DEBUG
LOGGER_AS_JSON=false
LOGGER_ENABLE_OLTP=true

OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
OTEL_SERVICE_NAME=storage_server
OTEL_ENVIRONMENT=development
OTEL_SERVICE_VERSION=1.0.0
OTEL_METRICS_PUSH_TIMEOUT=1s

RATE_LIMITER_LIMIT=100
RATE_LIMITER_PERIOD=1s
```

---

## Запуск

### Локально

```bash
# Из корня монорепозитория
cd services/storage
go run cmd/server/main.go
```

Или с кастомным `.env`:

```bash
go run cmd/server/main.go -config-path=.env.local
```

### Сборка

```bash
# Из корня репозитория (используется go.work)
go build ./services/storage/...

# Тесты storage-сервиса
make test-storage

# Или из директории сервиса
cd services/storage
go build -o bin/storage-server cmd/server/main.go
```

### Проверка gRPC (grpcurl)

```bash
# Health check
grpcurl -plaintext localhost:50053 opens3.storage.v1.DataStorageService/HealthCheck

# Reflection (список методов)
grpcurl -plaintext localhost:50053 list opens3.storage.v1.DataStorageService

# Delete object (unary)
grpcurl -plaintext -d '{"blob_id": "550e8400-e29b-41d4-a716-446655440000"}' \
  localhost:50053 opens3.storage.v1.DataStorageService/DeleteObject
```

### Multipart Smoke Test

Для быстрого ручного happy-path smoke test multipart upload:

```bash
go run ./services/storage/cmd/multipart-smoke
```

С кастомными параметрами:

```bash
go run ./services/storage/cmd/multipart-smoke \
  -addr localhost:50053 \
  -content-type text/plain \
  -part1 "hello " \
  -part2 "world" \
  -chunk-size 3
```

CLI создаёт multipart session, загружает 2 части несколькими чанками, выполняет `CompleteMultipartUpload`, затем читает итоговый blob обратно и проверяет содержимое и MD5.

---

## Хранилище на диске

```
DATA_DIR/                            (по умолчанию /data/blobs)
├── ab/
│   └── abcd1111-2222-3333-4444-555566667777
├── c4/
│   └── c4ef1111-2222-3333-4444-555566667777
└── ...

MULTIPART_DIR/                       (по умолчанию /data/staging)
├── completed/
│   ├── ab/
│   │   └── abcd1111-2222-3333-4444-555566667777.json
│   └── ...
└── uploads/
    ├── {upload_id}/
    │   ├── manifest.json            ← expected_parts + content_type + blob_id
    │   ├── part_00001
    │   ├── part_00002
    │   └── ...
    └── {blob_id}/
        ├── manifest.json            ← staging single-part upload
        └── object.bin
```

- `blob_id` — UUID v4, генерируется сервисом при `StoreObject`; финальный файл публикуется как `DATA_DIR/{blob_id[0:2]}/{blob_id}`
- `upload_id` — UUID v4, генерируется при `InitiateMultipartUpload`; multipart-части складываются в `MULTIPART_DIR/uploads/{upload_id}/`
- Single-part upload тоже проходит через staging: сначала создаётся `object.bin`, затем файл атомарно публикуется в `blobs/`
- После успешного `CompleteMultipartUpload` session cleanup выполняется best-effort; идемпотентность retry обеспечивается через `completed/{upload_id[0:2]}/{upload_id}.json`
- Для атомарной записи используются уникальные temp-файлы с последующим `os.Rename`, поэтому stale `*.tmp` после crash не должны ломать retry

---

## Dependency Injection

DI реализован через паттерн **Service Provider** с ленивой инициализацией.
Каждый компонент создаётся при первом обращении и кэшируется:

```
service_provider.go

StorageHandler(ctx)           ← создаёт handler при первом вызове
    └── StorageService(ctx)   ← создаёт service при первом вызове
        └── StorageRepository(ctx) ← создаёт repo при первом вызове
```

Файл: `internal/app/service_provider.go`

Цепочка зависимостей:
```
Handler → StorageService (interface) → StorageRepository (interface)
                                              ↓
                                        StorageConfig (DATA_DIR, MULTIPART_DIR)
```

---

## Обработка ошибок

Определены в `internal/errors/domain_errors/apperrors.go`:

| Ошибка | gRPC код | Когда |
|---|---|---|
| `ErrBlobNotFound` | `NOT_FOUND` | blob_id не существует на диске |
| `ErrUploadNotFound` | `NOT_FOUND` | upload_id multipart не найден |
| `ErrInvalidBlobSize` | `INVALID_ARGUMENT` | size < 0 или фактический размер не совпал с ожидаемым |
| `ErrInvalidUpload` | `INVALID_ARGUMENT` | некорректная multipart-сессия |
| `ErrInvalidParts` | `INVALID_ARGUMENT` | пустой/невалидный список частей или не совпало expected_parts |
| `ErrInvalidPartNumber` | `INVALID_ARGUMENT` | Некорректный номер части |
| `ErrChecksumMismatch` | `INVALID_ARGUMENT` | MD5 не совпадает при CompleteMultipart |
| `ErrDiskFull` | `RESOURCE_EXHAUSTED` | Недостаточно места на диске |
| `ErrInternal` | `INTERNAL` | Внутренняя ошибка сервиса |

Ошибки автоматически конвертируются в gRPC status codes через middleware
`validationInterceptor.ErrorCodesUnaryInterceptor` и `validationInterceptor.ErrorCodesStreamInterceptor` из shared kit.

Также есть transport-level ошибки, которые возникают ещё в handler/middleware:

- `INVALID_ARGUMENT` для нарушенного streaming contract (`size`, `content_type`, `upload_id`, `part_number` не в том сообщении)
- `CANCELED` если клиент отменил контекст
- `DEADLINE_EXCEEDED` если истёк deadline запроса
- `INTERNAL` для неожиданных ошибок FS и для пустого client stream, потому что `io.ErrUnexpectedEOF` сейчас не маппится в CommonError

---

## Observability

### Логирование

- Библиотека: [zap](https://github.com/uber-go/zap) (через `shared/pkg/go-kit/logger`)
- Формат: text (development) или JSON (production, `LOGGER_AS_JSON=true`)
- Экспорт в OTEL через OTLP (`LOGGER_ENABLE_OLTP=true`)

### Трейсинг

- OpenTelemetry SDK
- Экспорт через OTLP gRPC в Collector
- Каждый gRPC вызов автоматически создаёт span через `tracing.UnaryServerInterceptor`

### Метрики

- OpenTelemetry SDK
- Latency гистограммы и error rate для unary и streaming gRPC методов
- Автоматический сбор через `metricsInterceptor.MetricsInterceptor` и `metricsInterceptor.StreamMetricsInterceptor`
- Дополнительно для storage:
  - `storage_read_bytes_total`
  - `storage_write_bytes_total`
  - `storage_filesystem_usage_bytes`
  - `storage_data_dir_usage_bytes`

### Health Check

Эндпоинт `HealthCheck` возвращает:
- `SERVING` — сервис готов, `DATA_DIR` доступна
- `NOT_SERVING` — `DATA_DIR` недоступна или заканчивается место
- `UNKNOWN` — статус не определён

Также зарегистрирован стандартный gRPC Health Checking Protocol
(`grpc.health.v1.Health`) для использования с Kubernetes probes.

---

## Kafka-интеграция

> **Текущий статус:** не реализована (следующий этап: Phase 3)

Планируемая интеграция:

| Направление | Топик | Описание |
|---|---|---|
| **Producer** | `object-stored` | После успешного `StoreObject` — уведомление для Metadata Service |
| **Consumer** | `object-deleted` | Получение `blob_id` → удаление файла с диска (идемпотентно) |

---

## Разработка

### Пререквизиты

- Go 1.24.1+
- protoc + protoc-gen-go + protoc-gen-go-grpc (для генерации proto)

### Генерация protobuf

```bash
# Из корня репозитория
make generate-storage
```

Результат: `shared/pkg/go/storage/v1/storage.pb.go` + `storage_grpc.pb.go`

### Сборка и проверка

```bash
# Из корня репозитория (go.work подтягивает shared)
go build ./services/storage/...

# Unit + component tests
go test ./services/storage/...
```

### Генерация minimock для service tests

```bash
cd services/storage
PATH="$PWD/bin:$PATH" go generate ./pkg/mocks
```

### Структура модулей

```
go.work
├── ./shared                    # Shared kit + сгенерированные proto
├── ./services/storage          # Этот сервис
├── ./services/users
└── ./services/auth
```

Модуль `shared` подключается через `go.work`, а не через `require` в `go.mod`.
После пуша сгенерированных proto-файлов в remote `go mod tidy` отработает без ошибок.

### Добавление нового RPC

1. Добавить метод в `shared/api/storage/v1/storage.proto`
2. Запустить `make generate-storage`
3. Добавить метод в интерфейс `StorageRepository` (`repository/repository.go`) — если нужен новый метод FS
4. Реализовать в `repository/storage/`
5. Добавить метод в интерфейс `StorageService` (`service/service.go`)
6. Реализовать в `service/storage/`
7. Реализовать handler в `handler/storage/` (новый файл)

---

## TODO

Подробный план реализации с описанием всех фаз, взаимодействий и подводных камней:
[docs/storage-service-implementation-plan.md](../../docs/storage-service-implementation-plan.md)

- [x] Реализовать реальное чтение/запись на файловую систему в Repository
- [x] Подсчёт MD5 при записи blob
- [x] Range-запросы при чтении (RetrieveObject)
- [x] Multipart: хранение частей, склейка, отмена
- [ ] Kafka producer (`object-stored`) и consumer (`object-deleted`)
- [x] Проверка свободного места на диске в HealthCheck
- [ ] Шардирование файлов по первым байтам UUID
- [ ] TTL/garbage collection для зависших multipart-сессий
- [x] Unit/component-тесты для service и repository слоёв
- [x] Dockerfile
- [x] Интеграция в docker-compose.yml
