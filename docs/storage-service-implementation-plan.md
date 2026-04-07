# Storage Service — план реализации

Дата: 2026-04-07

---

## Текущее состояние

### Что реализовано

| Слой | Статус | Детали |
|------|--------|--------|
| Proto-контракт | ✅ Готов | 8 RPC методов, 18 типов сообщений |
| Handler (gRPC) | ✅ Готов | Все 8 методов, стриминг работает |
| Service | ✅ Готов | StoreObject с валидацией размера/content-type, multipart orchestration реализован |
| Repository | ✅ Готов | Реальная FS-запись/чтение/удаление blob и полный multipart lifecycle |
| Config | ✅ Готов | 6 групп настроек, env-парсинг |
| DI / App lifecycle | ✅ Готов | Service Provider, graceful shutdown, middleware |
| Dockerfile | ✅ Готов | Multi-stage Alpine build |
| docker-compose | ✅ Готов | Volume `storage-data:/data`, порт 50053 |
| Kafka | ❌ Нет | Нет ни producer, ни consumer |
| Unit/component-тесты | ✅ Готов | Repository, service и gRPC component tests покрывают blob и multipart flow |

### Что завершено по факту

**Repository:**
- `StoreBlob` пишет на диск через temporary file + `os.Rename`, считает MD5 и размер
- `RetrieveBlob` и `RetrieveBlobRange` читают реальные файлы, корректно обрабатывают range
- `DeleteBlob` идемпотентно удаляет blob
- `HealthCheck` проверяет доступность `DATA_DIR`, запись temp-файла и свободное место
- `CreateMultipartSession`, `StorePart`, `AssembleParts`, `CleanupMultipart` реализованы поверх `MULTIPART_DIR`

**Service:**
- `StoreObject` валидирует размер, нормализует `contentType`, удаляет blob при size mismatch
- Multipart flow больше не stub: service создаёт сессию, валидирует parts и делегирует filesystem-операции в repo
- На успешных write-операциях обновляются storage write metrics

---

## Кто вызывает Storage

Storage — это **«тупой» blob-хранилище**. Он не знает о бакетах, ключах, правах доступа или S3 API. Работает только с `blob_id` (UUID) и байтами.

### Синхронные вызовы (gRPC)

| Вызывающий | Метод Storage | Когда | Что получает |
|---|---|---|---|
| **Gateway** | `StoreObject` (stream) | PutObject — после Check на AuthZ | `blob_id`, `checksum_md5` |
| **Gateway** | `RetrieveObject` (stream) | GetObject — после GetObjectMeta от Metadata | stream байтов + `total_size` |
| **Gateway** | `DeleteObject` | **Не напрямую** — через Kafka (см. ниже) | — |
| **Gateway** | `InitiateMultipartUpload` | CreateMultipartUpload | `upload_id` |
| **Gateway** | `UploadPart` (stream) | UploadPart | `part_checksum_md5` |
| **Gateway** | `CompleteMultipartUpload` | CompleteMultipartUpload | `blob_id`, `checksum_md5` |
| **Gateway** | `AbortMultipartUpload` | AbortMultipartUpload | `success` |
| **Gateway** | `HealthCheck` | GET /ready | `ServingStatus` |

> **Важно:** Gateway — единственный потребитель Storage по gRPC. Никакой другой сервис не вызывает Storage напрямую.

### Асинхронные взаимодействия (Kafka)

| Направление | Топик | Producer | Consumer | Payload | Назначение |
|---|---|---|---|---|---|
| Storage → | `object-stored` | **Storage** | Metadata | `blob_id`, `checksum_md5`, `size_bytes` | Подтверждение записи blob (backup-путь для Metadata) |
| → Storage | `object-deleted` | **Metadata** | Storage, AuthZ | `blob_id` | Удаление файла с диска (идемпотентно) |

### Кого вызывает Storage

**Никого.** Storage не вызывает другие gRPC-сервисы. Взаимодействие только входящее:
- Входящий gRPC от Gateway
- Входящий Kafka от Metadata (`object-deleted`)
- Исходящий Kafka к Metadata (`object-stored`)

---

## Полные flow операций

### PutObject (простая загрузка)

```
Client → PUT /{bucket}/{key}
  │
  ▼ Gateway
  1. Извлечь user_id из JWT
  2. AuthZ.Check("user:{uid}", "write", "object:{bucket}")  → ALLOW
  3. Storage.StoreObject(stream bytes)                       → blob_id, checksum_md5
  4. Metadata.CreateObjectVersion(bucket, key, blob_id, size, etag)
  5. AuthZ.WriteTuple("bucket:{bucket}", "PARENT_OF", "object:{bucket}/{key}")
  6. → 200 OK { ETag, version_id }

  (async) Storage → Kafka: object-stored { blob_id }
```

### GetObject

```
Client → GET /{bucket}/{key}
  │
  ▼ Gateway
  1. AuthZ.Check("user:{uid}", "read", "object:{bucket}/{key}")  → ALLOW
  2. Metadata.GetObjectMeta(bucket, key)  → blob_id, size, etag, content_type
  3. Storage.RetrieveObject(blob_id, offset, length)          → stream bytes
  4. → 200 OK + body stream
```

### DeleteObject

```
Client → DELETE /{bucket}/{key}
  │
  ▼ Gateway
  1. AuthZ.Check("user:{uid}", "delete", "object:{bucket}/{key}")  → ALLOW
  2. Metadata.DeleteObjectMeta(bucket, key)  → blob_id
  3. → 204 No Content

  (async) Metadata → Kafka: object-deleted { blob_id }
           ├── Storage: удаляет /data/blobs/{blob_id}
           └── AuthZ: удаляет узел из графа
```

> **Подводный камень:** Gateway НЕ вызывает `Storage.DeleteObject` напрямую. Удаление blob происходит асинхронно через Kafka после того, как Metadata пометит объект удалённым. Это гарантирует, что метаданные обновляются атомарно, а физическое удаление — eventually consistent.

### Multipart Upload

```
Client → POST /{bucket}/{key}?uploads
  1. AuthZ.Check("user:{uid}", "write", "object:{bucket}")  → ALLOW
  2. Storage.InitiateMultipartUpload(expected_parts, content_type)  → upload_id
  3. → 200 OK { upload_id }

Client → PUT /{bucket}/{key}?partNumber=N&uploadId=X  (повторяется N раз)
  4. Storage.UploadPart(upload_id, part_number, stream bytes)  → part_checksum_md5
  5. → 200 OK { ETag: part_checksum_md5 }

Client → POST /{bucket}/{key}?uploadId=X
  6. Storage.CompleteMultipartUpload(upload_id, parts[])  → blob_id, checksum_md5
  7. Metadata.CreateObjectVersion(bucket, key, blob_id, size, etag)
  8. AuthZ.WriteTuple(...)
  9. → 200 OK { ETag, version_id }
```

---

## Что нужно реализовать

### Фаза 1 — Repository: реальная работа с файловой системой

Статус: ✅ завершена

Это **фундамент** — без него весь сервис бесполезен.

#### 1.1. `StoreBlob(ctx, reader) → (*BlobMeta, error)`

Что делает:
1. Генерирует `blob_id = uuid.New()`
2. Создаёт файл `{DATA_DIR}/{blob_id}` (или с шардированием: `{DATA_DIR}/{blob_id[0:2]}/{blob_id}`)
3. Копирует `reader` → файл, одновременно считая MD5 через `io.TeeReader` + `crypto/md5`
4. Считает `size_bytes` по мере записи
5. Возвращает `BlobMeta{ BlobID, ChecksumMD5, SizeBytes, ContentType }`

Подводные камни:
- **Атомарность записи:** писать во временный файл `{blob_id}.tmp`, потом `os.Rename` → предотвращает битые файлы при крэше
- **Проверка свободного места:** перед записью проверять `syscall.Statfs` на `DATA_DIR`
- **Права доступа:** `0644` для файлов, `0755` для поддиректорий

#### 1.2. `RetrieveBlob(ctx, blobID, offset, length) → (io.ReadCloser, int64, error)`

Что делает:
1. Открывает файл `{DATA_DIR}/{blob_id}`
2. Если `offset > 0` — `file.Seek(offset, io.SeekStart)`
3. Если `length > 0` — оборачивает в `io.LimitReader(file, length)`
4. Возвращает `file` (реализует `io.ReadCloser`), `totalSize` (из `file.Stat()`)

Подводные камни:
- **Файл не найден:** возвращать `ErrBlobNotFound` (→ gRPC `NOT_FOUND`)
- **Range за пределами файла:** не паниковать, обрезать до конца файла
- **Не забывать Close:** caller должен закрыть reader; handler уже делает это

#### 1.3. `DeleteBlob(ctx, blobID) → error`

Что делает:
1. `os.Remove({DATA_DIR}/{blob_id})`
2. Если файл не найден — **не ошибка** (идемпотентность)

Подводные камни:
- Минимум: идемпотентность — `os.IsNotExist(err)` → `nil`

#### 1.4. `HealthCheck(ctx) → error`

Что делает:
1. Проверяет, что `DATA_DIR` существует и доступен для записи (создать + удалить temp file)
2. Опционально: `syscall.Statfs` — вернуть ошибку если свободного места < порог

---

### Фаза 2 — Service: бизнес-логика и multipart

Статус: ✅ завершена

После того как repository работает с реальными файлами, в service-слое появляется настоящая логика.

#### 2.1. `StoreObject` — добавить валидацию

Текущий код:
```go
func (s *storageService) StoreObject(ctx context.Context, reader io.Reader, _ int64, _ string) (*model.BlobMeta, error) {
    return s.repo.StoreBlob(ctx, reader)
}
```

Что добавить:
- Проверка `size > 0` (если известен заранее)
- Проверка `contentType` не пустой (или fallback на `application/octet-stream`)
- Передавать `contentType` в repo для сохранения в метаданных BlobMeta

#### 2.2. Multipart — реализовать полный цикл

**`InitiateMultipartUpload(ctx, expectedParts, contentType) → (uploadID, error)`**
- Создать директорию `{MULTIPART_DIR}/{uploadID}/`
- Записать мета-файл (expectedParts, contentType, createdAt) — для валидации при Complete
- Нужен новый метод в repository: `CreateMultipartSession(ctx, uploadID, expectedParts) error`

**`UploadPart(ctx, uploadID, partNumber, reader) → (checksumMD5, error)`**
- Валидация: `partNumber >= 1`
- Валидация: директория `{MULTIPART_DIR}/{uploadID}` существует → иначе `ErrUploadNotFound`
- Записать файл `{MULTIPART_DIR}/{uploadID}/part_{partNumber}`
- Считать MD5 при записи
- Нужен новый метод в repository: `StorePart(ctx, uploadID, partNumber, reader) (string, error)`

**`CompleteMultipartUpload(ctx, uploadID, parts) → (*BlobMeta, error)`**
- Валидация: все заявленные части существуют на диске
- Валидация: MD5 каждой части совпадает с тем, что прислал клиент (→ `ErrChecksumMismatch`)
- Склеить части в один файл `{DATA_DIR}/{blob_id}` (по порядку part_number)
- Считать MD5 итогового файла
- Удалить директорию `{MULTIPART_DIR}/{uploadID}/`
- Нужен новый метод в repository: `AssembleParts(ctx, uploadID, parts, destBlobID) (*BlobMeta, error)`

**`AbortMultipartUpload(ctx, uploadID) → error`**
- Удалить директорию `{MULTIPART_DIR}/{uploadID}/` со всем содержимым
- Идемпотентно: если не найден — не ошибка
- Нужен новый метод в repository: `CleanupMultipart(ctx, uploadID) error`

#### Расширение интерфейса `StorageRepository`

```go
type StorageRepository interface {
    // Существующие методы
    StoreBlob(ctx context.Context, reader io.Reader) (*model.BlobMeta, error)
    RetrieveBlob(ctx context.Context, blobID string, rangeStart, rangeEnd int64) (io.ReadCloser, int64, error)
    DeleteBlob(ctx context.Context, blobID string) error
    HealthCheck(ctx context.Context) error

    // Новые методы для multipart
    CreateMultipartSession(ctx context.Context, uploadID string, expectedParts int32) error
    StorePart(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (checksumMD5 string, err error)
    AssembleParts(ctx context.Context, uploadID string, parts []model.PartInfo, destBlobID string) (*model.BlobMeta, error)
    CleanupMultipart(ctx context.Context, uploadID string) error
}
```

---

### Фаза 3 — Kafka-интеграция

Статус: ⏭️ следующий этап

Фаза 3 нужно проектировать уже не как "добавим один producer и один consumer", а как часть
полного object lifecycle в системе. Если целевая архитектура — **Metadata authoritative + pending/finalize**,
то Kafka нужна не для замены Metadata как источника истины, а для:

- надёжной межсервисной координации
- async cleanup
- repair/reconcile flow
- outbox-driven интеграций с AuthZ и Storage

#### 3.1. Какие топики нужны в системе

Ниже перечислены топики не только для Storage Service, а для всей системы object lifecycle.

| Топик | Тип | Producer | Consumer | Обязателен | Назначение |
|---|---|---|---|---|---|
| `object-delete-requested` | command | Metadata | Storage, AuthZ | ✅ | Объект логически удалён, нужно асинхронно почистить blob и графовые связи |
| `object-blob-stored` | event | Storage | Metadata | ⚠️ Желателен | Storage подтверждает, что blob/staging blob успешно записан и доступен для finalize/reconcile |
| `object-finalized` | event | Metadata | AuthZ, audit/reconcile workers | ✅ | Metadata зафиксировала объект как committed и сделала его видимым |
| `object-aborted` | event | Metadata | Storage, reconcile workers | ✅ | Upload session отменена или истекла, staging-данные можно очищать |
| `object-delete-confirmed` | event | Storage | Metadata, reconcile workers | ⚠️ Желателен | Storage подтверждает физическое удаление blob |
| `bucket-deleted` | event | Metadata | AuthZ, cleanup workers | ✅ | Бакет удалён из metadata, нужно чистить внешние проекции |

#### 3.2. Семантика топиков

##### `object-delete-requested`

Главная команда удаления blob из Storage.

Когда публикуется:
- Metadata завершила логическое удаление объекта
- объект больше не должен быть доступен через `GET/HEAD/List`

Минимальный payload:

```json
{
  "event_id": "uuid",
  "object_id": "uuid",
  "version_id": "uuid",
  "blob_id": "uuid",
  "bucket_id": "uuid",
  "bucket_name": "photos",
  "object_key": "2026/cat.jpg",
  "occurred_at": 1775606400000
}
```

Что делают consumers:
- Storage удаляет blob идемпотентно
- AuthZ удаляет объект и связанные отношения из своей проекции

##### `object-blob-stored`

Это событие не делает объект опубликованным само по себе. Оно подтверждает, что Storage успешно записал blob
или staging-данные и их можно использовать для finalize/reconcile.

Когда публикуется:
- после успешного `StoreObject`
- после успешного `CompleteMultipartUpload`

Минимальный payload:

```json
{
  "event_id": "uuid",
  "upload_id": "uuid",
  "blob_id": "uuid",
  "checksum_md5": "d41d8cd9...",
  "size_bytes": 1048576,
  "occurred_at": 1775606400000
}
```

Использование:
- Metadata может сверять pending uploads и blob writes
- reconcile worker может находить orphan staging/final blobs

##### `object-finalized`

Событие публикации объекта как committed в authoritative metadata plane.

Когда публикуется:
- после успешного `FinalizeUpload` / commit version в Metadata

Минимальный payload:

```json
{
  "event_id": "uuid",
  "object_id": "uuid",
  "version_id": "uuid",
  "blob_id": "uuid",
  "bucket_id": "uuid",
  "bucket_name": "photos",
  "object_key": "2026/cat.jpg",
  "etag": "d41d8cd9...",
  "size_bytes": 1048576,
  "content_type": "image/jpeg",
  "occurred_at": 1775606400000
}
```

Использование:
- AuthZ может создать/обновить проекцию объектного ресурса
- audit и indexer-процессы получают единый authoritative event
- downstream-системы узнают о committed object, а не о факте записи байт

##### `object-aborted`

Событие очистки незавершённой upload session.

Когда публикуется:
- пользователь вызвал AbortMultipartUpload
- pending upload протух по TTL
- Metadata перевела upload в `failed/expired/aborted`

Минимальный payload:

```json
{
  "event_id": "uuid",
  "upload_id": "uuid",
  "bucket_id": "uuid",
  "bucket_name": "photos",
  "object_key": "2026/cat.jpg",
  "reason": "expired",
  "occurred_at": 1775606400000
}
```

Использование:
- Storage удаляет staging-файлы
- reconcile worker закрывает зависшие workflow

##### `object-delete-confirmed`

Подтверждение физического удаления blob.

Когда публикуется:
- Storage реально удалил blob
- или установил, что blob уже отсутствует

Использование:
- Metadata может завершить internal cleanup и пометить blob как purged
- reconcile worker видит завершённый delete flow

#### 3.3. Как эти топики используются в системе целиком

##### PutObject / CompleteMultipartUpload

Основной commit path:
1. Gateway создаёт upload intent через Metadata
2. Gateway пишет байты в Storage
3. Storage публикует `object-blob-stored`
4. Gateway вызывает finalize в Metadata
5. Metadata в своей транзакции фиксирует новую версию
6. Metadata публикует `object-finalized`

Важно:
- объект становится видимым только после шага 5
- `object-blob-stored` — это сигнал о durable write, но не о публикации объекта

##### DeleteObject

1. Gateway вызывает delete в Metadata
2. Metadata логически удаляет объект или создаёт delete marker
3. Metadata публикует `object-delete-requested`
4. Storage удаляет blob
5. Storage публикует `object-delete-confirmed`

Важно:
- пользователь больше не видит объект уже после шага 2
- физическое удаление происходит асинхронно

##### Abort / TTL cleanup

1. Metadata переводит upload в `aborted` или `expired`
2. Metadata публикует `object-aborted`
3. Storage чистит staging-данные

#### 3.4. Что должен реализовать именно Storage Service в Фазе 3

Обязательный минимум для Storage:

1. Consumer `object-delete-requested`
2. Producer `object-blob-stored`
3. Опционально producer `object-delete-confirmed`
4. Idempotent delete handler
5. Kafka lifecycle в `app.go`

Где добавлять:
- producer в service-слой
- consumer в отдельный пакет `internal/consumer/`
- запуск consumer параллельно с gRPC сервером из `app.go`

Новая зависимость service-слоя:

```go
type storageService struct {
    repo     repository.StorageRepository
    producer kafka.Producer
}
```

#### 3.5. Подводные камни Kafka

- **At-least-once delivery:** delete и cleanup handlers должны быть идемпотентными
- **Outbox нужен на стороне Metadata:** commit object и публикация `object-finalized` должны быть связаны через одну локальную транзакционную границу
- **`object-blob-stored` не должен считаться publish-событием:** иначе появится двойной источник истины
- **Ordering:** если критичен порядок по одному объекту, партиционировать по `object_id` или `blob_id`
- **DLQ:** для poison messages и систематических ошибок нужен отдельный dead-letter topic
- **Graceful shutdown:** consumer должен закрываться вместе с gRPC сервером

#### 3.6. Новые env vars

```env
KAFKA_BOOTSTRAP=kafka:9092
KAFKA_OBJECT_BLOB_STORED_TOPIC=object-blob-stored
KAFKA_OBJECT_DELETE_REQUESTED_TOPIC=object-delete-requested
KAFKA_OBJECT_DELETE_CONFIRMED_TOPIC=object-delete-confirmed
KAFKA_OBJECT_FINALIZED_TOPIC=object-finalized
KAFKA_OBJECT_ABORTED_TOPIC=object-aborted
KAFKA_BUCKET_DELETED_TOPIC=bucket-deleted
KAFKA_CONSUMER_GROUP=storage-consumer
```

Примечание:
- для ближайшей реализации Storage Service достаточно начать с `object-delete-requested` и `object-blob-stored`
- остальные топики стоит зафиксировать уже сейчас как целевую архитектуру, чтобы не зацементировать слишком узкую модель `object-stored/object-deleted`

---

### Фаза 4 — Hardening

- **Шардирование файлов:** `{DATA_DIR}/{blob_id[0:2]}/{blob_id}` — предотвращает >100k файлов в одной директории
- **Очистка зависших multipart:** фоновый goroutine, удаляющий `{MULTIPART_DIR}/{uploadID}` старше N часов
- **Метрики:** размер записанного/прочитанного в bytes (OTEL counter)
- **Rate limiter per-method:** разные лимиты для StoreObject (тяжёлый) vs HealthCheck (лёгкий)

Примечание:
- На текущем этапе `CompleteMultipartUpload` не удаляет multipart-сессию при recoverable ошибках сборки
  (например, missing part или checksum mismatch), чтобы не ломать retry flow.
  Очистка таких зависших сессий должна решаться через TTL-based garbage collection.

---

## Подводные камни и неочевидности

### 1. Gateway НЕ вызывает Storage.DeleteObject напрямую

Это самая частая ошибка при чтении архитектуры. Удаление blob — **асинхронное** через Kafka. Gateway вызывает Metadata, Metadata публикует событие, Storage потребляет и удаляет файл.

Следствие: `DeleteObject` gRPC метод нужен **только** для ручного/административного удаления, не для основного flow.

### 2. `StoreObject` не знает ключ объекта

Storage получает только байты и возвращает `blob_id`. Gateway потом связывает `blob_id` с ключом через Metadata. Storage принципиально не знает, к какому бакету/объекту относится blob.

### 3. `ChecksumMD5` — ответственность Storage

MD5 считается **на стороне Storage** при записи, а не на стороне Gateway. Gateway передаёт байты стримом, Storage пишет и считает MD5 одновременно через `io.TeeReader`.

### 4. Range-запросы в RetrieveObject

`offset = 0, length = 0` означает «весь файл». `length = 0` при `offset > 0` означает «читать до конца файла начиная с offset`. Если `offset + length > file_size`, обрезать до конца файла. Это критично для S3-совместимости (заголовок `Range: bytes=0-`).

### 5. Multipart: порядок частей не гарантирован

Части могут приходить не по порядку (`part_3` раньше `part_1`). При `CompleteMultipartUpload` нужно склеивать строго по `part_number`, а не по порядку загрузки.

### 6. Multipart: части можно перезаписывать

По S3-спецификации, повторная загрузка `UploadPart` с тем же `part_number` перезаписывает предыдущую. Это легально.

### 7. `PARENT_OF` пока не работает в AuthZ

Как зафиксировано в [wiki-audit](wiki-audit-2026-04-02.md): AuthZ **не** traverses `PARENT_OF` при `Check()`. Это значит, что права на бакет **не** наследуются объектами автоматически. Gateway должен назначать права явно на каждый объект, либо AuthZ нужно доработать. Это архитектурное решение за пределами Storage, но влияет на flow.

### 8. Конфликт `object-stored` и синхронного CreateObjectVersion

В текущем flow Gateway **синхронно** вызывает `Metadata.CreateObjectVersion` после `Storage.StoreObject`. Kafka-событие `object-stored` — это **backup-путь**, не основной. Нужно решить: 
- Metadata вызывается только через Kafka (fully async)?
- Или Gateway вызывает Metadata синхронно, а Kafka — для надёжности?

Текущий дизайн в CLAUDE.md — **синхронный** вызов от Gateway, Kafka как backup.

---

## Какие тесты нужны

### Unit-тесты (мокаем зависимости, быстрые, в CI)

#### Repository-слой

| Тест | Что проверяет |
|---|---|
| `TestStoreBlob_Success` | Файл создан на диске, MD5 корректный, size совпадает |
| `TestStoreBlob_DiskFull` | Корректная ошибка при нехватке места |
| `TestStoreBlob_AtomicWrite` | При ошибке записи tmp-файл удаляется |
| `TestRetrieveBlob_Success` | Содержимое файла возвращается корректно |
| `TestRetrieveBlob_NotFound` | `ErrBlobNotFound` при несуществующем blob_id |
| `TestRetrieveBlob_Range` | Частичное чтение (rangeStart/rangeEnd) |
| `TestDeleteBlob_Success` | Файл удалён с диска |
| `TestDeleteBlob_Idempotent` | Повторное удаление — не ошибка |
| `TestHealthCheck_DirAccessible` | Возвращает nil для доступной директории |
| `TestHealthCheck_DirMissing` | Возвращает ошибку для недоступной директории |

> Эти тесты работают с `os.MkdirTemp` — реальная FS, но в изолированной tmp-директории. Формально это не "чистые" unit-тесты, но они быстрые и не требуют внешних зависимостей.

#### Service-слой

| Тест | Что проверяет |
|---|---|
| `TestStoreObject_Validation` | Валидация contentType/size перед вызовом repo |
| `TestInitiateMultipart_CreatesSession` | Repo.CreateMultipartSession вызван с правильными аргументами |
| `TestUploadPart_InvalidPartNumber` | Ошибка при partNumber < 1 |
| `TestUploadPart_UploadNotFound` | ErrUploadNotFound прокидывается |
| `TestCompleteMultipart_ChecksumMismatch` | ErrChecksumMismatch при несовпадении MD5 |
| `TestCompleteMultipart_Success` | Все части собраны, repo.AssembleParts вызван |
| `TestAbortMultipart_Idempotent` | Не ошибка если upload не найден |

> Эти тесты мокают `StorageRepository` через minimock и уже реализованы для blob + multipart сценариев.

#### Kafka consumer (после реализации Фазы 3)

| Тест | Что проверяет |
|---|---|
| `TestObjectDeletedHandler_Success` | repo.DeleteBlob вызван с корректным blob_id |
| `TestObjectDeletedHandler_AlreadyDeleted` | Не ошибка, offset коммитится |
| `TestObjectDeletedHandler_InvalidPayload` | Некорректное сообщение логируется, не крэшит consumer |

### Интеграционные тесты (реальная FS, без моков)

Тестируют repository-слой с реальной файловой системой.

| Тест | Что проверяет |
|---|---|
| `TestStoreAndRetrieve` | Записать blob → прочитать → содержимое совпадает |
| `TestStoreAndDelete` | Записать → удалить → RetrieveBlob возвращает NotFound |
| `TestMultipartFullCycle` | Initiate → Upload 3 parts → Complete → Retrieve → данные корректны |
| `TestMultipartAbort` | Initiate → Upload parts → Abort → директория удалена |
| `TestConcurrentStores` | 10 параллельных StoreBlob → все успешны, файлы не перемешаны |
| `TestLargeBlob` | Запись/чтение blob > 100MB (проверяет стриминг, а не буферизацию в память) |

> Запускаются с `go test -tags=integration` или в отдельном CI-job. Используют `os.MkdirTemp`, не требуют Docker.

### E2E тесты (весь стек, требуют Docker)

Тестируют gRPC API через реальный gRPC-клиент.

| Тест | Что проверяет |
|---|---|
| `TestStoreObject_E2E` | gRPC StoreObject → blob на диске → RetrieveObject → данные совпадают |
| `TestDeleteObject_E2E` | StoreObject → DeleteObject → RetrieveObject → NOT_FOUND |
| `TestMultipart_E2E` | Полный multipart цикл через gRPC |
| `TestHealthCheck_E2E` | HealthCheck → SERVING |
| `TestStoreObject_RateLimit` | 100+ запросов → rate limiter отвечает RESOURCE_EXHAUSTED |

> Запускаются через `docker compose up storage` + gRPC-клиент из тестов. Требуют CI с Docker.

### Приоритет по тестам

1. ✅ Repository tests с реальной FS
2. ✅ Service tests с minimock для blob и multipart
3. ✅ Component tests через gRPC для blob и multipart
4. ⏭️ После Фазы 3: unit-тесты Kafka producer/consumer
5. ⏭️ Phase 4: дополнительные E2E и hardening-сценарии
## Порядок реализации (рекомендация)

```
Фаза 1: Repository FS          ← можно делать сейчас, независимо от всех
  │
  ├── 1a. StoreBlob (атомарная запись + MD5)
  ├── 1b. RetrieveBlob (range support)
  ├── 1c. DeleteBlob (идемпотентный)
  └── 1d. HealthCheck (проверка DATA_DIR)
  │
  └── Интеграционные тесты repository
  │
Фаза 2: Multipart               ← зависит от Фазы 1
  │
  ├── 2a. Расширить StorageRepository (4 новых метода)
  ├── 2b. Реализовать multipart в repository
  ├── 2c. Реализовать multipart в service (убрать стабы)
  └── 2d. Unit + интеграционные тесты multipart
  │
Фаза 3: Kafka                   ← зависит от Фазы 1, параллельна с Фазой 2
  │
  ├── 3a. Добавить kafka-зависимость (sarama или segmentio/kafka-go)
  ├── 3b. Producer: object-stored в StoreObject/CompleteMultipart
  ├── 3c. Consumer: object-deleted → DeleteBlob
  └── 3d. Unit-тесты consumer
  │
Фаза 4: Hardening               ← после всего
  │
  ├── 4a. Шардирование файлов на диске ({blob_id[0:2]}/...)
  ├── 4b. Очистка зависших multipart
  ├── 4c. E2E тесты
  └── 4d. Метрики bytes read/written
  │
Фаза 5: Распределённое хранение  → [distributed-storage-architecture.md](distributed-storage-architecture.md)
  │
Фаза 6: Kubernetes + облако      → [kubernetes-deployment-plan.md](kubernetes-deployment-plan.md)
